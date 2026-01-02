"""
美学评分接口路由
"""

from typing import List

from fastapi import APIRouter, HTTPException

from train.model import get_score_level
from ..models import (
    AestheticData,
    AestheticRequest,
    AestheticResponse,
    ErrorResponse,
)
from ..services import model_service

router = APIRouter(tags=["aesthetics"])


@router.post(
    "/aesthetics",
    response_model=AestheticResponse,
    responses={
        400: {"model": ErrorResponse},
        500: {"model": ErrorResponse},
    },
)
async def evaluate_aesthetics(request: AestheticRequest) -> AestheticResponse:
    """
    评估图片美学质量

    基于 SigLIP2 + 自训练 LoRA 模型
    评分范围: 1-10，输出 10 类评分概率分布，加权平均得到最终分数

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
            AestheticData(
                index=i,
                score=round(result.score, 2),
                level=get_score_level(result.score),
                distribution=(
                    result.distribution.tolist() if request.return_distribution else None
                ),
            )
            for i, result in enumerate(results)
        ]

        return AestheticResponse(
            data=data,
            backend=model_service.backend.backend_type if model_service.backend else "unknown",
        )

    except FileNotFoundError as e:
        raise HTTPException(status_code=400, detail=f"File not found: {e}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
