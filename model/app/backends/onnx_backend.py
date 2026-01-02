# -*- coding: utf-8 -*-
"""
ONNX 推理后端实现

支持美学评分和向量嵌入的 ONNX 推理
"""

import os
from pathlib import Path
from typing import List, Optional

import numpy as np
from PIL import Image

from .base import AestheticResult, BackendType, BaseBackend

# 获取项目根目录
PROJECT_ROOT = Path(__file__).parent.parent.parent


def softmax_numpy(x: np.ndarray, axis: int = -1) -> np.ndarray:
    """计算 softmax (NumPy 版本)"""
    exp_x = np.exp(x - np.max(x, axis=axis, keepdims=True))
    return exp_x / np.sum(exp_x, axis=axis, keepdims=True)


def distribution_to_score_numpy(distribution: np.ndarray) -> float:
    """将概率分布转换为加权平均分数"""
    scores = np.arange(1, 11, dtype=distribution.dtype)
    if distribution.ndim == 1:
        return float((distribution * scores).sum())
    else:
        return float((distribution * scores).sum(axis=-1))


def _get_providers(device: Optional[str]) -> List[str]:
    """根据设备选择 ONNX provider"""
    if device == "cuda":
        return ["CUDAExecutionProvider", "CPUExecutionProvider"]
    elif device == "coreml":
        return ["CoreMLExecutionProvider", "CPUExecutionProvider"]
    else:
        return ["CPUExecutionProvider"]


class ONNXBackend(BaseBackend):
    """ONNX 推理后端实现

    支持美学评分和向量嵌入推理
    """

    def __init__(self):
        super().__init__()
        self.aesthetic_session = None  # 美学评分 ONNX session
        self.embedding_session = None  # 向量嵌入 ONNX session
        self.text_session = None  # 文本嵌入 ONNX session

    @property
    def is_loaded(self) -> bool:
        return self.aesthetic_session is not None

    @property
    def backend_type(self) -> BackendType:
        return BackendType.ONNX

    def initialize(
        self,
        device: str = "auto",
        base_model_path: Optional[str] = None,
        lora_weights_path: Optional[str] = None,
        onnx_model_path: Optional[str] = None,
    ) -> None:
        if self.is_loaded:
            print("ONNX backend already loaded, skipping initialization")
            return

        import onnxruntime as ort

        self.device = device or "cpu"

        # 设置默认路径
        if base_model_path is None:
            base_model_path = str(PROJECT_ROOT / "siglip2")
        if onnx_model_path is None:
            onnx_model_path = str(PROJECT_ROOT / "train" / "model.onnx")

        # ONNX 模型路径配置
        aesthetic_onnx_path = onnx_model_path
        embedding_onnx_path = str(PROJECT_ROOT / "train" / "siglip_vision.onnx")
        text_onnx_path = str(PROJECT_ROOT / "train" / "siglip_text.onnx")

        print(f"Initializing ONNX backend...")
        print(f"  Device: {self.device}")
        print(f"  Base model: {base_model_path}")

        # 加载处理器 (使用 transformers)
        from transformers import AutoProcessor

        self.processor = AutoProcessor.from_pretrained(
            base_model_path,
            trust_remote_code=True,
            use_fast=True
        )

        # 配置 ONNX session
        sess_options = ort.SessionOptions()
        sess_options.intra_op_num_threads = 4
        sess_options.inter_op_num_threads = 4
        sess_options.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL

        # 选择 provider
        providers = _get_providers(device)

        # 加载美学评分模型
        if os.path.exists(aesthetic_onnx_path):
            print(f"  Aesthetic ONNX: {aesthetic_onnx_path}")
            self.aesthetic_session = ort.InferenceSession(
                aesthetic_onnx_path,
                sess_options,
                providers=providers,
            )
        else:
            print(f"  Warning: Aesthetic ONNX not found: {aesthetic_onnx_path}")

        # 加载图像嵌入模型
        if os.path.exists(embedding_onnx_path):
            print(f"  Vision ONNX: {embedding_onnx_path}")
            self.embedding_session = ort.InferenceSession(
                embedding_onnx_path,
                sess_options,
                providers=providers,
            )
        else:
            print(f"  Warning: Vision ONNX not found: {embedding_onnx_path}")

        # 加载文本嵌入模型
        if os.path.exists(text_onnx_path):
            print(f"  Text ONNX: {text_onnx_path}")
            self.text_session = ort.InferenceSession(
                text_onnx_path,
                sess_options,
                providers=providers,
            )
        else:
            print(f"  Warning: Text ONNX not found: {text_onnx_path}")

        print("ONNX backend loaded successfully!")

    def infer_aesthetic(self, images: List[Image.Image]) -> List[AestheticResult]:
        if self.aesthetic_session is None:
            raise RuntimeError("Aesthetic ONNX model not loaded.")

        if not images:
            return []

        # 预处理
        pixel_values = (
            self.processor(images=images, return_tensors="pt")
            .pixel_values.numpy()
            .astype(np.float32)
        )

        # 推理
        outputs = self.aesthetic_session.run(None, {"pixel_values": pixel_values})
        logits = outputs[0]
        distributions = softmax_numpy(logits, axis=-1)

        # 构建结果
        results = []
        for i in range(len(images)):
            dist = distributions[i] if distributions.ndim > 1 else distributions
            score = distribution_to_score_numpy(dist)
            results.append(AestheticResult(score=score, distribution=dist))

        return results

    def infer_image_embedding(self, images: List[Image.Image]) -> List[np.ndarray]:
        if self.embedding_session is None:
            raise RuntimeError(
                "Vision ONNX model not loaded. "
                "Please export siglip_vision.onnx first."
            )

        if not images:
            return []

        # 预处理
        pixel_values = (
            self.processor(images=images, return_tensors="pt")
            .pixel_values.numpy()
            .astype(np.float32)
        )

        # 推理
        outputs = self.embedding_session.run(None, {"pixel_values": pixel_values})
        embeddings = outputs[0]

        # 归一化
        norms = np.linalg.norm(embeddings, axis=-1, keepdims=True)
        embeddings = embeddings / norms

        return [embeddings[i] for i in range(len(images))]

    def infer_text_embedding(self, texts: List[str]) -> List[np.ndarray]:
        if self.text_session is None:
            raise RuntimeError(
                "Text ONNX model not loaded. "
                "Please export siglip_text.onnx first."
            )

        if not texts:
            return []

        # 预处理
        inputs = self.processor(
            text=texts,
            return_tensors="pt",
            padding=True,
            truncation=True,
        )

        # 转换为 numpy
        input_ids = inputs["input_ids"].numpy()
        attention_mask = inputs.get("attention_mask")
        if attention_mask is not None:
            attention_mask = attention_mask.numpy()

        # 推理
        onnx_inputs = {"input_ids": input_ids}
        if attention_mask is not None:
            onnx_inputs["attention_mask"] = attention_mask

        outputs = self.text_session.run(None, onnx_inputs)
        embeddings = outputs[0]

        # 归一化
        norms = np.linalg.norm(embeddings, axis=-1, keepdims=True)
        embeddings = embeddings / norms

        return [embeddings[i] for i in range(len(texts))]
