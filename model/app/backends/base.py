# -*- coding: utf-8 -*-
"""
推理后端抽象接口定义

定义所有后端必须实现的接口
"""

from abc import ABC, abstractmethod
from enum import Enum
from typing import List, NamedTuple, Optional

import numpy as np
from PIL import Image


class AestheticResult(NamedTuple):
    """美学评分结果"""
    score: float
    distribution: np.ndarray  # 10 类概率分布


class BackendType(str, Enum):
    """推理后端类型"""
    PYTORCH = "pytorch"
    ONNX = "onnx"


class BaseBackend(ABC):
    """推理后端抽象基类

    所有后端实现必须继承此类并实现所有抽象方法
    """

    def __init__(self):
        self.device: Optional[str] = None
        self.processor = None
        self.hidden_size = 1152
        self.num_classes = 10

    @property
    @abstractmethod
    def is_loaded(self) -> bool:
        """检查模型是否已加载"""
        pass

    @property
    @abstractmethod
    def backend_type(self) -> BackendType:
        """返回后端类型"""
        pass

    @abstractmethod
    def initialize(
        self,
        device: str = "auto",
        base_model_path: Optional[str] = None,
        lora_weights_path: Optional[str] = None,
        onnx_model_path: Optional[str] = None,
    ) -> None:
        """初始化模型

        Args:
            device: 运行设备 ('cuda', 'cpu', 'mps')，None 则自动选择
            base_model_path: 基础模型路径
            lora_weights_path: LoRA 权重路径 (PyTorch 后端使用)
            onnx_model_path: ONNX 模型路径 (ONNX 后端使用)
        """
        pass

    @abstractmethod
    def infer_aesthetic(self, images: List[Image.Image]) -> List[AestheticResult]:
        """批量美学评分推理

        Args:
            images: PIL Image 对象列表

        Returns:
            AestheticResult 列表，包含评分和概率分布
        """
        pass

    @abstractmethod
    def infer_image_embedding(self, images: List[Image.Image]) -> List[np.ndarray]:
        """获取图片嵌入向量

        使用 SigLIP 模型获取归一化的图像嵌入，
        与文本嵌入在同一向量空间中，适合用于图文相似度搜索。

        Args:
            images: PIL Image 对象列表

        Returns:
            归一化的嵌入向量列表，每个向量维度为 1152
        """
        pass

    @abstractmethod
    def infer_text_embedding(self, texts: List[str]) -> List[np.ndarray]:
        """获取文本嵌入向量

        使用 SigLIP 模型获取归一化的文本嵌入，
        与图像嵌入在同一向量空间中，适合用于图文相似度搜索。

        Args:
            texts: 文本字符串列表

        Returns:
            归一化的嵌入向量列表，每个向量维度为 1152
        """
        pass
