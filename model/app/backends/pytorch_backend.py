# -*- coding: utf-8 -*-
"""
PyTorch 推理后端实现

基于 SigLIP2 + 自训练 LoRA 美学评分模型
"""

import os
import sys
from pathlib import Path
from typing import List, Optional

import numpy as np
import torch
import torch.nn.functional as F
from PIL import Image
from transformers import AutoModel, AutoProcessor, SiglipModel, SiglipProcessor

from .base import AestheticResult, BackendType, BaseBackend

# 获取项目根目录并添加 train 目录到路径
PROJECT_ROOT = Path(__file__).parent.parent.parent
TRAIN_DIR = PROJECT_ROOT / "train"
if str(TRAIN_DIR) not in sys.path:
    sys.path.insert(0, str(TRAIN_DIR))

from train.model import AestheticLoRAModel, distribution_to_score_numpy


class PyTorchBackend(BaseBackend):
    """PyTorch 推理后端实现"""

    def __init__(self):
        super().__init__()
        self.dtype = None
        self.torch_model = None
        self.siglip_model: Optional[SiglipModel] = None
        self.siglip_processor: Optional[SiglipProcessor] = None

    @property
    def is_loaded(self) -> bool:
        return self.torch_model is not None

    @property
    def backend_type(self) -> BackendType:
        return BackendType.PYTORCH

    def initialize(
            self,
            device: str = "auto",
            base_model_path: Optional[str] = None,
            lora_weights_path: Optional[str] = None,
            onnx_model_path: Optional[str] = None,
    ) -> None:
        if self.is_loaded:
            print("PyTorch backend already loaded, skipping initialization")
            return
        self.device = device

        # 设置默认路径
        if base_model_path is None:
            base_model_path = str(PROJECT_ROOT / "siglip2")
        if lora_weights_path is None:
            lora_weights_path = str(PROJECT_ROOT / "train" / "best_lora.pth")

        print(f"Initializing PyTorch backend...")
        print(f"  Device: {device}")
        print(f"  Base model: {base_model_path}")
        print(f"  LoRA weights: {lora_weights_path}")

        # 确定数据类型
        if device == "cuda" and torch.cuda.is_bf16_supported():
            self.dtype = torch.bfloat16
        elif device == "mps":
            self.dtype = torch.bfloat16
        else:
            self.dtype = torch.float32
        print(f"  Dtype: {self.dtype}")

        # 加载处理器
        self.processor = AutoProcessor.from_pretrained(
            base_model_path,
            trust_remote_code=True,
            use_fast=True
        )

        # 初始化美学评分模型
        self._initialize_aesthetic_model(base_model_path, lora_weights_path, device)

        # 初始化 SigLIP 基础模型用于向量编码
        self._initialize_siglip_model(base_model_path, device)

        print("PyTorch backend loaded successfully!")

    def _initialize_aesthetic_model(
            self, base_model_path: str, lora_weights_path: str, device: str
    ) -> None:
        """初始化美学评分模型"""
        from peft import LoraConfig, TaskType, get_peft_model

        # 加载权重配置
        if not os.path.exists(lora_weights_path):
            raise FileNotFoundError(f"LoRA weights not found: {lora_weights_path}")

        checkpoint = torch.load(lora_weights_path, map_location="cpu")
        if "config" not in checkpoint or "state_dict" not in checkpoint:
            raise ValueError("Invalid weight file format")

        config = checkpoint["config"]
        state_dict = checkpoint["state_dict"]
        print(f"  LoRA config: r={config['lora_r']}, alpha={config['lora_alpha']}")

        # 加载基础模型
        full_model = AutoModel.from_pretrained(
            base_model_path,
            trust_remote_code=True,
        )

        # 获取 vision_model
        if hasattr(full_model, "vision_model"):
            base_model = full_model.vision_model
        else:
            base_model = full_model

        # 配置 LoRA
        lora_config = LoraConfig(
            r=config["lora_r"],
            lora_alpha=config["lora_alpha"],
            target_modules=config["lora_target_modules"],
            lora_dropout=config.get("lora_dropout", 0.0),
            bias="none",
            task_type=TaskType.FEATURE_EXTRACTION,
        )
        lora_model = get_peft_model(base_model, lora_config)

        # 创建完整模型
        self.torch_model = AestheticLoRAModel(
            base_model=lora_model,
            hidden_size=self.hidden_size,
            dropout=0.0,
            num_classes=self.num_classes,
        )

        # 加载权重
        self.torch_model.load_state_dict(state_dict, strict=False)
        self.torch_model = self.torch_model.to(device).to(self.dtype)
        self.torch_model.eval()

    def _initialize_siglip_model(self, base_model_path: str, device: str) -> None:
        """初始化 SigLIP 基础模型用于向量编码"""
        print("  Loading SigLIP model for embedding...")
        self.siglip_processor = SiglipProcessor.from_pretrained(base_model_path, use_fast=True)
        self.siglip_model = SiglipModel.from_pretrained(base_model_path)
        self.siglip_model = self.siglip_model.to(self.dtype).to(device)
        self.siglip_model.eval()

    def infer_aesthetic(self, images: List[Image.Image]) -> List[AestheticResult]:
        if not self.is_loaded:
            raise RuntimeError("Model not loaded. Call initialize() first.")

        if not images:
            return []

        # 预处理
        pixel_values = (
            self.processor(images=images, return_tensors="pt")
            .pixel_values.to(self.dtype)
            .to(self.device)
        )

        # 推理
        with torch.inference_mode():
            logits = self.torch_model(pixel_values)
            distributions = F.softmax(logits, dim=-1).float().cpu().numpy()

        # 构建结果
        results = []
        for i in range(len(images)):
            dist = distributions[i] if distributions.ndim > 1 else distributions
            score = distribution_to_score_numpy(dist)
            results.append(AestheticResult(score=score, distribution=dist))

        return results

    def infer_image_embedding(self, images: List[Image.Image]) -> List[np.ndarray]:
        if not self.is_loaded:
            raise RuntimeError("Model not loaded. Call initialize() first.")

        if self.siglip_model is None or self.siglip_processor is None:
            raise RuntimeError("SigLIP model not loaded for image encoding.")

        if not images:
            return []

        # 1. 安全转换：确保所有图片都是 RGB 模式，避免 RGBA/灰度图导致维度错误或色彩异常
        rgb_images = [img.convert("RGB") for img in images]

        # 2. 预处理
        inputs = self.siglip_processor(
            images=rgb_images,
            return_tensors="pt",
        )

        # 3. 关键修正：将 pixel_values 显式转换为模型的 dtype (bfloat16/float16)
        # 原始代码只做了 to(device)，会导致 float32 输入进 bfloat16 模型
        inputs = {
            k: v.to(self.device).to(self.dtype) if v.dtype.is_floating_point else v.to(self.device)
            for k, v in inputs.items()
        }

        with torch.inference_mode():
            # 获取图像特征
            image_features = self.siglip_model.get_image_features(**inputs)

            # 归一化 (SigLIP/CLIP 必须步骤)
            image_features = image_features / image_features.norm(dim=-1, keepdim=True)

            # 转回 float32 再存入 numpy，防止数据库驱动不支持 bf16
            embeddings = image_features.float().cpu().numpy()

        return [embeddings[i] for i in range(len(images))]

    def infer_text_embedding(self, texts: List[str]) -> List[np.ndarray]:
        if not self.is_loaded:
            raise RuntimeError("Model not loaded. Call initialize() first.")

        if self.siglip_model is None or self.siglip_processor is None:
            raise RuntimeError("SigLIP model not loaded for text encoding.")

        if not texts:
            return []

        inputs = self.siglip_processor(
            text=texts,
            return_tensors="pt",
            padding="max_length", # 建议：显式指定 padding 策略，有时默认不仅是 padding=True
            truncation=True,
        )

        # 文本输入的 input_ids 是整数，不需要转 dtype，保持原样即可
        inputs = {k: v.to(self.device) for k, v in inputs.items()}

        with torch.inference_mode():
            text_features = self.siglip_model.get_text_features(**inputs)
            text_features = text_features / text_features.norm(dim=-1, keepdim=True)
            embeddings = text_features.float().cpu().numpy()

        return [embeddings[i] for i in range(len(texts))]

# 在 initialize() 之后运行这段测试代码
# def sanity_check(backend):
#     print("--- Running Sanity Check ---")
#
#     # 1. 准备测试数据
#     test_text = ["a cat", "a dog", "a flower"]
#     # 找一张你要测试的图片（比如之前的猫）
#     try:
#         img = Image.open("/Users/wuyuejiang/Downloads/pipeline-cat-chonk.jpeg")
#     except:
#         # 如果没有图，生成一张纯黑图测试代码跑通，但无法测试相关性
#         img = Image.new('RGB', (224, 224), color='black')
#         print("Warning: Using dummy image")
#
#     # 2. 获取向量
#     img_emb = backend.infer_image_embedding([img])[0]  # Shape: (D,)
#     txt_embs = backend.infer_text_embedding(test_text)  # List of (D,)
#
#     # 3. 计算余弦相似度 (点积，因为已经归一化了)
#     print(f"Image vs Texts Similarity:")
#     for text, t_emb in zip(test_text, txt_embs):
#         sim = np.dot(img_emb, t_emb)
#         print(f"  '{text}': {sim:.4f}")
# #
# #
# # backend = PyTorchBackend()
# # backend.initialize("mps",
# #                    "/Users/wuyuejiang/IdeaProjects/gallary/model/siglip2",
# #                    "/Users/wuyuejiang/IdeaProjects/gallary/model/train/best_lora.pth"
# #                    "",
# #                    )
# # sanity_check(backend)
# # # 4. 判断逻辑
# # # 如果是猫图，"a cat" 的分数应该最高，且通常 > 0.1 (SigLIP 的 raw score 可能不像 CLIP 那么高，但相对顺序必须对)
# # # 如果所有分数都非常接近（例如都是 0.001 或 0.999），说明模型坍塌或 dtype 错误。
