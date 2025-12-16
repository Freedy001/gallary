"""
模型推理服务
封装 Aesthetic Predictor V2.5 和 SigLIP 嵌入功能
"""

import base64
import io
import os
import re
import ssl
from typing import List, NamedTuple, Optional

import numpy as np
import requests
import torch
from PIL import Image

# 禁用 SSL 验证（用于 HuggingFace 镜像）
ssl._create_default_https_context = ssl._create_unverified_context

# 设置 HuggingFace 镜像
HF_MIRROR = os.environ.get("HF_ENDPOINT", "https://hf-mirror.com")
os.environ["HF_ENDPOINT"] = HF_MIRROR


class InferenceResult(NamedTuple):
    """推理结果"""
    score: float
    embedding: np.ndarray


class ModelService:
    """模型推理服务 - 单例模式"""

    _instance: Optional["ModelService"] = None
    _initialized: bool = False

    def __new__(cls) -> "ModelService":
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def __init__(self):
        if self._initialized:
            return

        self.model = None
        self.preprocessor = None
        self.device = None
        # 用于文本编码的 SigLIP 模型
        self.siglip_model = None
        self.siglip_processor = None
        self._initialized = True

    def initialize(self, device: Optional[str] = None) -> None:
        """
        初始化模型

        Args:
            device: 运行设备 ('cuda', 'cpu', 'mps')，None 则自动选择
        """
        if self.model is not None:
            return

        # 自动选择设备
        if device is None:
            if torch.cuda.is_available():
                device = "cuda"
            elif torch.backends.mps.is_available():
                device = "mps"
            else:
                device = "cpu"
        self.device = device

        print(f"Loading model on device: {device}")
        print(f"Using HuggingFace mirror: {HF_MIRROR}")

        # 导入模型加载函数
        from aesthetic_predictor_v2_5 import convert_v2_5_from_siglip

        siglip_model_name = "google/siglip-so400m-patch14-384"

        model, preprocessor = convert_v2_5_from_siglip(
            encoder_model_name=siglip_model_name,
            low_cpu_mem_usage=True,
            trust_remote_code=True,
        )

        self.preprocessor = preprocessor
        self.model = model.to(torch.bfloat16).to(self.device)
        self.model.eval()

        # 加载 SigLIP 基础模型用于文本编码
        # 注意：SiglipProcessor 需要使用在线模型名称以确保 tokenizer 文件完整
        print("Loading SigLIP model for text encoding...")
        from transformers import SiglipProcessor, SiglipModel

        self.siglip_processor = SiglipProcessor.from_pretrained(siglip_model_name)
        self.siglip_model = SiglipModel.from_pretrained(siglip_model_name)
        self.siglip_model = self.siglip_model.to(torch.bfloat16).to(self.device)
        self.siglip_model.eval()

        print("Model loaded successfully!")

    @property
    def is_loaded(self) -> bool:
        """检查模型是否已加载"""
        return self.model is not None

    def load_image(self, input_str: str) -> Image.Image:
        """
        加载图片，支持多种输入格式

        Args:
            input_str: 图片输入，支持:
                - 本地文件路径
                - HTTP/HTTPS URL
                - Base64 编码 (支持 data:image/xxx;base64,... 格式)

        Returns:
            PIL Image 对象
        """
        # 检查是否为 Base64 编码
        if input_str.startswith("data:image"):
            # data:image/jpeg;base64,/9j/4AAQ...
            match = re.match(r"data:image/[^;]+;base64,(.+)", input_str)
            if match:
                base64_data = match.group(1)
                image_bytes = base64.b64decode(base64_data)
                return Image.open(io.BytesIO(image_bytes)).convert("RGB")

        # 检查是否为纯 Base64（无前缀）
        if self._is_base64(input_str):
            image_bytes = base64.b64decode(input_str)
            return Image.open(io.BytesIO(image_bytes)).convert("RGB")

        # 检查是否为 URL
        if input_str.startswith(("http://", "https://")):
            response = requests.get(input_str, timeout=30)
            response.raise_for_status()
            return Image.open(io.BytesIO(response.content)).convert("RGB")

        # 作为本地路径处理
        return Image.open(input_str).convert("RGB")

    def _is_base64(self, s: str) -> bool:
        """检查字符串是否为有效的 Base64 编码"""
        # Base64 字符串通常较长且不包含路径分隔符
        if len(s) < 100:
            return False
        if "/" in s and not s.startswith("data:"):
            # 可能是路径
            if os.path.exists(s):
                return False
        try:
            # 尝试解码前 100 个字符
            base64.b64decode(s[:100], validate=True)
            return True
        except Exception:
            return False

    def infer(self, image: Image.Image) -> InferenceResult:
        """
        对单张图片进行推理

        Args:
            image: PIL Image 对象

        Returns:
            InferenceResult(score, embedding)
        """
        if not self.is_loaded:
            raise RuntimeError("Model not loaded. Call initialize() first.")

        # 预处理图片
        pixel_values = (
            self.preprocessor(images=image, return_tensors="pt")
            .pixel_values.to(torch.bfloat16)
            .to(self.device)
        )

        # 推理
        with torch.inference_mode():
            output = self.model(pixel_values)
            score = output.logits.squeeze().float().cpu().numpy()
            embedding = output.hidden_states.squeeze().float().cpu().numpy()

        return InferenceResult(
            score=float(score),
            embedding=embedding
        )

    def infer_batch(self, images: List[Image.Image]) -> List[InferenceResult]:
        """
        批量推理

        Args:
            images: PIL Image 对象列表

        Returns:
            InferenceResult 列表
        """
        if not self.is_loaded:
            raise RuntimeError("Model not loaded. Call initialize() first.")

        if not images:
            return []

        # 批量预处理
        pixel_values = (
            self.preprocessor(images=images, return_tensors="pt")
            .pixel_values.to(torch.bfloat16)
            .to(self.device)
        )

        # 批量推理
        with torch.inference_mode():
            output = self.model(pixel_values)
            scores = output.logits.squeeze(-1).float().cpu().numpy()
            embeddings = output.hidden_states.float().cpu().numpy()

        # 处理单张图片的情况
        if len(images) == 1:
            scores = [scores.item() if scores.ndim == 0 else scores[0]]
            embeddings = [embeddings] if embeddings.ndim == 1 else [embeddings[0]]

        return [
            InferenceResult(score=float(s), embedding=e)
            for s, e in zip(scores, embeddings)
        ]

    def infer_text(self, texts: List[str]) -> List[np.ndarray]:
        """
        对文本进行嵌入推理

        Args:
            texts: 文本字符串列表

        Returns:
            嵌入向量列表
        """
        if not self.is_loaded:
            raise RuntimeError("Model not loaded. Call initialize() first.")

        if self.siglip_model is None or self.siglip_processor is None:
            raise RuntimeError("SigLIP model not loaded for text encoding.")

        # 使用 SigLIP 的文本编码器
        inputs = self.siglip_processor(
            text=texts,
            return_tensors="pt",
            padding=True,
            truncation=True
        )
        inputs = {k: v.to(self.device) for k, v in inputs.items()}

        with torch.inference_mode():
            text_features = self.siglip_model.get_text_features(**inputs)
            # 归一化 (与图片嵌入保持一致的处理方式)
            text_features = text_features / text_features.norm(dim=-1, keepdim=True)
            embeddings = text_features.float().cpu().numpy()

        return [embeddings[i] for i in range(len(texts))]

    def infer_image_embedding(self, images: List[Image.Image]) -> List[np.ndarray]:
        """
        对图片进行嵌入推理（仅返回嵌入向量，不返回美学评分）

        Args:
            images: PIL Image 对象列表

        Returns:
            嵌入向量列表
        """
        if not self.is_loaded:
            raise RuntimeError("Model not loaded. Call initialize() first.")

        if not images:
            return []

        # 使用现有的批量推理
        results = self.infer_batch(images)
        return [r.embedding for r in results]


def get_score_level(score: float) -> str:
    """
    根据分数返回等级描述
    V2.5 评分标准：5.5+ 被认为是高美学分数
    """
    if score >= 7.5:
        return "优秀 (Excellent)"
    elif score >= 6.5:
        return "很好 (Very Good)"
    elif score >= 5.5:
        return "良好 (Good)"
    elif score >= 4.5:
        return "一般 (Average)"
    elif score >= 3.5:
        return "较差 (Below Average)"
    else:
        return "差 (Poor)"


# 全局服务实例
model_service = ModelService()
