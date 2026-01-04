"""
Pydantic 模型定义
兼容 OpenAI Embeddings API 格式 和 阿里云 Multimodal-Embedding API 格式
"""

from typing import Dict, List, Optional, Union

from pydantic import BaseModel, Field


# ============== 嵌入接口模型 (兼容 OpenAI) ==============

class EmbeddingRequest(BaseModel):
    """嵌入请求 - 兼容 OpenAI 格式"""
    model: str = Field(
        default="siglip2-so400m-patch16-512",
        description="模型名称"
    )
    input: Union[str, List[str]] = Field(
        ...,
        description="图片输入，支持本地路径、URL 或 Base64 编码"
    )
    encoding_format: str = Field(
        default="float",
        description="编码格式: float 或 base64"
    )


class EmbeddingData(BaseModel):
    """单个嵌入结果"""
    object: str = "embedding"
    index: int
    embedding: List[float]


class EmbeddingUsage(BaseModel):
    """Token 使用统计"""
    prompt_tokens: int
    total_tokens: int


class EmbeddingResponse(BaseModel):
    """嵌入响应 - 兼容 OpenAI 格式"""
    object: str = "list"
    data: List[EmbeddingData]
    model: str
    usage: EmbeddingUsage


# ============== 美学评分接口模型 ==============

class AestheticRequest(BaseModel):
    """美学评分请求"""
    input: Union[str, List[str]] = Field(
        ...,
        description="图片输入，支持本地路径、URL 或 Base64 编码"
    )
    return_distribution: bool = Field(
        default=False,
        description="是否返回 10 类评分概率分布"
    )


class AestheticData(BaseModel):
    """单个美学评分结果"""
    index: int
    score: float = Field(..., description="美学评分 (1-10 加权平均)")
    level: str = Field(..., description="评分等级描述")
    distribution: Optional[List[float]] = Field(
        default=None,
        description="10 类评分概率分布 (仅当 return_distribution=true 时返回)"
    )


class AestheticResponse(BaseModel):
    """美学评分响应"""
    data: List[AestheticData]
    model: str = "siglip2-aesthetic-lora"
    backend: str = Field(default="pytorch", description="推理后端 (pytorch 或 onnx)")


# ============== 通用模型 ==============

class HealthResponse(BaseModel):
    """健康检查响应"""
    status: str = "ok"
    model_loaded: bool
    device: str
    backend: str = Field(default="pytorch", description="推理后端")


class ErrorResponse(BaseModel):
    """错误响应"""
    error: str
    detail: Optional[str] = None


# ============== 阿里云 Multimodal-Embedding 兼容模型 ==============

class MultimodalInput(BaseModel):
    """多模态输入 - 兼容阿里云格式"""
    contents: List[Dict[str, str]] = Field(
        ...,
        description="内容列表，每个元素为 {'text': '...'} 或 {'image': '...'}"
    )


class MultimodalEmbeddingRequest(BaseModel):
    """多模态嵌入请求 - 兼容阿里云格式"""
    model: str = Field(
        default="siglip2-so400m-patch16-512",
        description="模型名称"
    )
    input: MultimodalInput = Field(
        ...,
        description="输入内容"
    )


class MultimodalEmbeddingItem(BaseModel):
    """单个嵌入结果 - 兼容阿里云格式"""
    index: int = Field(..., description="结果索引")
    embedding: List[float] = Field(..., description="嵌入向量")
    type: str = Field(..., description="输入类型: text 或 image")


class MultimodalEmbeddingOutput(BaseModel):
    """嵌入输出 - 兼容阿里云格式"""
    embeddings: List[MultimodalEmbeddingItem]


class MultimodalEmbeddingUsage(BaseModel):
    """Token 使用统计 - 兼容阿里云格式"""
    input_tokens: int = Field(default=0, description="输入文本 Token 数")
    image_tokens: int = Field(default=0, description="图片 Token 数")


class MultimodalEmbeddingResponse(BaseModel):
    """多模态嵌入响应 - 兼容阿里云格式"""
    output: MultimodalEmbeddingOutput
    usage: MultimodalEmbeddingUsage
    model: str


# ============== 设置接口模型 ==============

class PromptOptimizerSettings(BaseModel):
    """提示词优化器设置（用于 API 请求参数）"""
    enabled: bool = Field(default=True, description="是否启用提示词优化")
    system_prompt: str = Field(
        default="",
        description="系统提示词，为空则使用默认提示词"
    )
