# -*- coding: utf-8 -*-
"""
美学评分模型定义

包含:
- AestheticMLP: 评分预测头
- AestheticLoRAModel: LoRA 美学评分模型
- 工具函数: distribution_to_score, softmax, get_score_level
"""

from typing import Tuple

import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F


class AestheticMLP(nn.Module):
    """美学评分分布预测头

    输出 10 个类别的概率分布，对应评分 1-10
    """

    def __init__(self, hidden_size: int = 1152, dropout: float = 0.1, num_classes: int = 10):
        super().__init__()
        self.num_classes = num_classes
        self.mlp = nn.Sequential(
            nn.Linear(hidden_size, 512),
            nn.GELU(),
            nn.Dropout(dropout),
            nn.Linear(512, 128),
            nn.GELU(),
            nn.Dropout(dropout),
            nn.Linear(128, num_classes),
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.mlp(x)


class AestheticLoRAModel(nn.Module):
    """LoRA 美学评分模型

    输出 10 类评分的概率分布
    """

    def __init__(
            self,
            base_model: nn.Module,
            hidden_size: int = 1152,
            dropout: float = 0.1,
            num_classes: int = 10,
    ):
        super().__init__()
        self.vision_model = base_model
        self.aesthetic_head = AestheticMLP(hidden_size, dropout, num_classes)
        self.num_classes = num_classes

    def forward(self, pixel_values: torch.Tensor) -> torch.Tensor:
        """返回 logits"""
        # PEFT 包装后需要通过 get_base_model() 获取底层模型
        if hasattr(self.vision_model, 'get_base_model'):
            base = self.vision_model.get_base_model()
            outputs = base(pixel_values=pixel_values)
        else:
            outputs = self.vision_model(pixel_values=pixel_values)

        hidden_states = outputs.last_hidden_state

        # Mean Pooling
        pooled_features = hidden_states.mean(dim=1)
        logits = self.aesthetic_head(pooled_features)

        return logits

    def forward_with_embedding(self, pixel_values: torch.Tensor) -> Tuple[torch.Tensor, torch.Tensor]:
        """返回 logits 和 embedding"""
        if hasattr(self.vision_model, 'get_base_model'):
            base = self.vision_model.get_base_model()
            outputs = base(pixel_values=pixel_values)
        else:
            outputs = self.vision_model(pixel_values=pixel_values)

        hidden_states = outputs.last_hidden_state
        pooled_features = hidden_states.mean(dim=1)
        logits = self.aesthetic_head(pooled_features)
        return logits, pooled_features

    def predict_distribution(self, pixel_values: torch.Tensor) -> torch.Tensor:
        """预测概率分布"""
        logits = self.forward(pixel_values)
        return F.softmax(logits, dim=-1)

    def predict_score(self, pixel_values: torch.Tensor) -> torch.Tensor:
        """预测加权平均分数"""
        prob = self.predict_distribution(pixel_values)
        return distribution_to_score_torch(prob)


def distribution_to_score_torch(distribution: torch.Tensor) -> torch.Tensor:
    """将概率分布转换为加权平均分数 (PyTorch 版本)

    Args:
        distribution: (batch_size, 10) 或 (10,) - 概率分布
    Returns:
        score: 加权平均分数 (1-10)
    """
    scores = torch.arange(1, 11, dtype=distribution.dtype, device=distribution.device)

    if distribution.dim() == 1:
        return (distribution * scores).sum()
    else:
        return (distribution * scores.unsqueeze(0)).sum(dim=1)


def distribution_to_score_numpy(distribution: np.ndarray) -> float:
    """将概率分布转换为加权平均分数 (NumPy 版本)

    Args:
        distribution: (10,) 或 (batch_size, 10) - 概率分布
    Returns:
        score: 加权平均分数 (1-10)
    """
    scores = np.arange(1, 11, dtype=distribution.dtype)
    if distribution.ndim == 1:
        return float((distribution * scores).sum())
    else:
        return float((distribution * scores).sum(axis=-1))


def softmax_numpy(x: np.ndarray, axis: int = -1) -> np.ndarray:
    """计算 softmax (NumPy 版本)"""
    exp_x = np.exp(x - np.max(x, axis=axis, keepdims=True))
    return exp_x / np.sum(exp_x, axis=axis, keepdims=True)


def get_score_level(score: float) -> str:
    """根据分数返回等级描述"""
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


def format_distribution(distribution: np.ndarray) -> str:
    """格式化概率分布为字符串"""
    bars = []
    for i, p in enumerate(distribution):
        bar_len = int(p * 40)  # 最大 40 个字符
        bar = "█" * bar_len
        bars.append(f"  {i + 1:2d}: {p:5.1%} {bar}")
    return "\n".join(bars)
