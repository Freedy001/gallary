"""
嵌入接口路由 - 兼容 OpenAI Embeddings API
"""

from typing import List, Union

from fastapi import APIRouter, HTTPException

from ..models import (
    EmbeddingData,
    EmbeddingRequest,
    EmbeddingResponse,
    EmbeddingUsage,
    ErrorResponse,
)
from ..services import model_service

router = APIRouter(tags=["embeddings"])


@router.post(
    "/embeddings",
    response_model=EmbeddingResponse,
    responses={
        400: {"model": ErrorResponse},
        500: {"model": ErrorResponse},
    },
)
async def create_embeddings(request: EmbeddingRequest) -> EmbeddingResponse:
    """
    创建图片嵌入向量 - 兼容 OpenAI Embeddings API

    支持的输入格式:
    - 本地文件路径: "/path/to/image.jpg"
    - HTTP URL: "https://example.com/image.jpg"
    - Base64: "data:image/jpeg;base64,/9j/4AAQ..." 或纯 Base64 字符串
    """
    if not model_service.is_loaded:
        raise HTTPException(status_code=500, detail="Model not loaded")

    # 统一输入为列表
    inputs: List[str] = (
        request.input if isinstance(request.input, list) else [request.input]
    )

    if not inputs:
        raise HTTPException(status_code=400, detail="Input cannot be empty")

    try:
        # 加载图片
        images = []
        for input_str in inputs:
            image = model_service.load_image(input_str)
            images.append(image)

        # 批量推理
        results = model_service.infer_batch(images)

        # 构建响应
        data = [
            EmbeddingData(
                index=i,
                embedding=result.embedding.tolist(),
            )
            for i, result in enumerate(results)
        ]

        return EmbeddingResponse(
            data=data,
            model=request.model,
            usage=EmbeddingUsage(
                prompt_tokens=len(inputs),
                total_tokens=len(inputs),
            ),
        )

    except FileNotFoundError as e:
        raise HTTPException(status_code=400, detail=f"File not found: {e}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
