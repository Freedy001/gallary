"""
Image Aesthetic & Embedding Service
基于 SigLIP2 + 自训练 LoRA 的图片美学评分与向量嵌入微服务

启动方式:
    python main.py                              # 默认 PyTorch 后端
    BACKEND=onnx python main.py                 # 使用 ONNX 后端
    PORT=8080 python main.py                    # 自定义端口
    uvicorn main:app --host 0.0.0.0 --port 8100
"""

import os
from contextlib import asynccontextmanager

import torch
import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.models import HealthResponse
from app.routers import aesthetics_router, embeddings_router, multimodal_embedding_router
from app.services import model_service, BackendType


@asynccontextmanager
async def lifespan(_):
    """应用生命周期管理 - 启动时加载模型"""
    # 获取配置
    device = os.environ.get("DEVICE", None)
    if device is None:
        if torch.cuda.is_available():
            device = "cuda"
        elif torch.backends.mps.is_available():
            device = "mps"
        else:
            device = "cpu"

    backend_str = os.environ.get("BACKEND", "pytorch").lower()
    backend = BackendType.ONNX if backend_str == "onnx" else BackendType.PYTORCH

    # 模型路径配置
    base_model_path = os.environ.get("BASE_MODEL_PATH", None)
    lora_weights_path = os.environ.get("LORA_WEIGHTS_PATH", None)
    onnx_model_path = os.environ.get("ONNX_MODEL_PATH", None)

    # 初始化模型
    model_service.initialize(
        device=device,
        backend=backend,
        base_model_path=base_model_path,
        lora_weights_path=lora_weights_path,
        onnx_model_path=onnx_model_path,
    )
    yield
    # 关闭时清理（如果需要）


# 创建 FastAPI 应用
app = FastAPI(
    title="Image Aesthetic & Embedding Service",
    description="""
基于 SigLIP2 + 自训练 LoRA 的图片美学评分与向量嵌入微服务

## 功能

- **嵌入接口** (`/v1/embeddings`): 兼容 OpenAI Embeddings API，生成图片向量
- **美学评分** (`/v1/aesthetics`): 评估图片美学质量，返回 1-10 分评分及概率分布
- **多模态嵌入** (`/v1/multimodal-embedding`): 兼容阿里云 API，支持文本和图片

## 模型

- 基础模型: google/siglip2-so400m-patch16-512
- 美学评分: 自训练 LoRA 模型，输出 10 类评分概率分布

## 后端

支持两种推理后端:
- **PyTorch**: 完整功能，支持 GPU 加速
- **ONNX**: 轻量化 CPU 推理，适合部署

## 输入格式

支持多种图片输入格式:
- 本地文件路径: `/path/to/image.jpg`
- HTTP URL: `https://example.com/image.jpg`
- Base64: `data:image/jpeg;base64,/9j/4AAQ...` 或纯 Base64 字符串
    """,
    version="2.0.0",
    lifespan=lifespan,
)

# CORS 中间件
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 注册路由
app.include_router(embeddings_router, prefix="/v1")
app.include_router(aesthetics_router, prefix="/v1")
app.include_router(multimodal_embedding_router, prefix="/v1")


@app.get("/health", response_model=HealthResponse, tags=["health"])
async def health_check() -> HealthResponse:
    """健康检查"""
    return HealthResponse(
        status="ok",
        model_loaded=model_service.is_loaded,
        device=model_service.device or "not initialized",
        backend=model_service.backend.backend_type if model_service.backend else "not initialized",
    )


@app.get("/", tags=["root"])
async def root():
    """根路径 - API 信息"""
    return {
        "name": "Image Aesthetic & Embedding Service",
        "version": "2.0.0",
        "model": {
            "base": "google/siglip2-so400m-patch16-512",
            "aesthetic": "siglip2-aesthetic-lora",
            "backend": model_service.backend.backend_type if model_service.backend else "not initialized",
        },
        "endpoints": {
            "embeddings": "/v1/embeddings",
            "aesthetics": "/v1/aesthetics",
            "multimodal_embedding": "/v1/multimodal-embedding",
            "health": "/health",
            "docs": "/docs",
        },
        "prompt_optimizer": {
            "enabled": prompt_optimizer_service.is_loaded,
            "model": "Qwen/Qwen3-0.6B",
        },
    }


if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8100))
    host = os.environ.get("HOST", "0.0.0.0")

    print(f"Starting server on {host}:{port}")
    uvicorn.run(app, host=host, port=port)
