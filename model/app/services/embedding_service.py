# -*- coding: utf-8 -*-
"""
模型推理服务

基于 SigLIP2 + 自训练 LoRA 美学评分模型
支持 PyTorch 和 ONNX 双后端
"""

import base64
import io
import os
import re
import ssl
from pathlib import Path
from typing import List, Optional

import numpy as np
import requests
from PIL import Image

from ..backends import (
    AestheticResult,
    BackendType,
    BaseBackend,
    ONNXBackend,
    PyTorchBackend,
)

# 禁用 SSL 验证（用于 HuggingFace 镜像）
ssl._create_default_https_context = ssl._create_unverified_context

# 设置 HuggingFace 镜像
HF_MIRROR = os.environ.get("HF_ENDPOINT", "https://hf-mirror.com")
os.environ["HF_ENDPOINT"] = HF_MIRROR

# 获取项目根目录
PROJECT_ROOT = Path(__file__).parent.parent


class ModelService:
    """模型推理服务 - 单例模式，支持 PyTorch 和 ONNX 双后端"""

    _instance: Optional["ModelService"] = None
    _initialized: bool = False

    def __new__(cls) -> "ModelService":
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def __init__(self):
        if self._initialized:
            return

        self.backend: Optional[BaseBackend] = None
        self._initialized = True

    def initialize(
            self,
            device: Optional[str] = None,
            backend: BackendType = BackendType.PYTORCH,
            base_model_path: Optional[str] = None,
            lora_weights_path: Optional[str] = None,
            onnx_model_path: Optional[str] = None,
    ) -> None:
        """
        初始化模型

        Args:
            device: 运行设备 ('cuda', 'cpu', 'mps')，None 则自动选择
            backend: 推理后端类型 (pytorch 或 onnx)
            base_model_path: 基础模型路径 (默认: siglip2/)
            lora_weights_path: LoRA 权重路径 (默认: train/best_lora.pth)
            onnx_model_path: ONNX 模型路径 (默认: train/model.onnx)
        """
        if self.is_loaded:
            print("Model already loaded, skipping initialization")
            return

        print(f"Initializing model service...")
        print(f"  Backend: {backend.value}")
        print(f"  Using HuggingFace mirror: {HF_MIRROR}")

        # 创建对应后端
        if backend == BackendType.PYTORCH:
            self.backend = PyTorchBackend()
        else:
            self.backend = ONNXBackend()

        # 初始化后端
        self.backend.initialize(
            device=device,
            base_model_path=base_model_path,
            lora_weights_path=lora_weights_path,
            onnx_model_path=onnx_model_path,
        )

        print("Model service initialized!")

    @property
    def is_loaded(self) -> bool:
        """检查模型是否已加载"""
        return self.backend is not None and self.backend.is_loaded

    @property
    def device(self) -> Optional[str]:
        """获取当前设备"""
        return self.backend.device if self.backend else None

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
            match = re.match(r"data:image/[^;]+;base64,(.+)", input_str)
            if match:
                base64_data = match.group(1)
                image_bytes = base64.b64decode(base64_data)
                return Image.open(io.BytesIO(image_bytes)).convert("RGB")

        # 检查是否为纯 Base64
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
        if len(s) < 100:
            return False
        if "/" in s and not s.startswith("data:"):
            if os.path.exists(s):
                return False
        try:
            base64.b64decode(s[:100], validate=True)
            return True
        except Exception:
            return False

    def infer(self, image: Image.Image) -> AestheticResult:
        """
        对单张图片进行美学评分推理

        Args:
            image: PIL Image 对象

        Returns:
            AestheticResult(score, distribution)
        """
        results = self.infer_batch([image])
        return results[0]

    def infer_batch(self, images: List[Image.Image]) -> List[AestheticResult]:
        """
        批量美学评分推理

        Args:
            images: PIL Image 对象列表

        Returns:
            AestheticResult 列表
        """
        if not self.is_loaded:
            raise RuntimeError("Model not loaded. Call initialize() first.")

        return self.backend.infer_aesthetic(images)

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
        for text in texts:
            print("计算文本向量：" + text)
        return self.backend.infer_text_embedding(texts)

    def infer_image_embedding(self, images: List[Image.Image]) -> List[np.ndarray]:
        """
        使用 SigLIP 模型获取图片嵌入向量（用于图片搜索）

        注意：返回的是归一化后的图像嵌入，与文本嵌入在同一向量空间中，
        适合用于图文相似度搜索。

        Args:
            images: PIL Image 对象列表

        Returns:
            归一化的嵌入向量列表，每个向量维度为 1152
        """
        if not self.is_loaded:
            raise RuntimeError("Model not loaded. Call initialize() first.")
        for text in images:
            print("计算图片向量：" + str(text.size))
        return self.backend.infer_image_embedding(images)


# 全局服务实例
model_service = ModelService()
