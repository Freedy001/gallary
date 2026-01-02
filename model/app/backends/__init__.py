# -*- coding: utf-8 -*-
"""
推理后端模块

支持 PyTorch 和 ONNX 双后端
"""

from .base import BackendType, BaseBackend, AestheticResult
from .onnx_backend import ONNXBackend
from .pytorch_backend import PyTorchBackend

__all__ = [
    "BackendType",
    "BaseBackend",
    "AestheticResult",
    "PyTorchBackend",
    "ONNXBackend",
]
