"""
多模态嵌入接口 - 兼容阿里云 Multimodal-Embedding API 格式
支持文本和图片的向量嵌入
文本查询会自动通过 Qwen3-0.6B 优化为更适合 SigLIP 的英文描述
"""

from fastapi import APIRouter, HTTPException

from ..models import (
    MultimodalEmbeddingRequest,
    MultimodalEmbeddingResponse,
    MultimodalEmbeddingOutput,
    MultimodalEmbeddingItem,
    MultimodalEmbeddingUsage,
)
from ..services import model_service

router = APIRouter(tags=["multimodal-embedding"])


@router.post("/multimodal-embedding", response_model=MultimodalEmbeddingResponse)
async def create_multimodal_embedding(
        req: MultimodalEmbeddingRequest,
) -> MultimodalEmbeddingResponse:
    """
    创建多模态嵌入向量 - 兼容阿里云 API 格式

    支持输入:
    - {"text": "文本内容"}
    - {"image": "图片路径/URL/Base64"}

    返回格式与阿里云 Multimodal-Embedding API 一致
    """
    if not model_service.is_loaded:
        raise HTTPException(status_code=503, detail="Model not loaded")

    contents = req.input.contents
    if not contents:
        raise HTTPException(status_code=400, detail="Input contents cannot be empty")

    embeddings_result: list[MultimodalEmbeddingItem] = []
    input_tokens = 0
    image_tokens = 0

    try:
        # 分离文本和图片内容
        texts = []
        text_indices = []
        images = []
        image_indices = []

        for idx, content in enumerate(contents):
            if "text" in content:
                texts.append(content["text"])
                text_indices.append(idx)
            elif "image" in content:
                image = model_service.load_image(content["image"])
                images.append(image)
                image_indices.append(idx)
            else:
                raise HTTPException(
                    status_code=400,
                    detail=f"Invalid content at index {idx}: must contain 'text' or 'image' key",
                )

        # 处理文本嵌入（先通过 Qwen3 优化提示词）
        if texts:
            # 确定是否启用提示词优化及使用哪个配置
            # 优先使用请求中的配置，否则使用服务端默认配置
            text_embeddings = model_service.infer_text(texts)
            for i, (text_idx, embedding) in enumerate(zip(text_indices, text_embeddings)):
                embeddings_result.append(
                    MultimodalEmbeddingItem(
                        index=text_idx,
                        embedding=embedding.tolist(),
                        type="text",
                    )
                )
            input_tokens = len(texts)

        # 处理图片嵌入
        if images:
            image_embeddings = model_service.infer_image_embedding(images)
            for i, (img_idx, embedding) in enumerate(zip(image_indices, image_embeddings)):
                embeddings_result.append(
                    MultimodalEmbeddingItem(
                        index=img_idx,
                        embedding=embedding.tolist(),
                        type="image",
                    )
                )
            image_tokens = len(images)

        # 按原始索引排序
        embeddings_result.sort(key=lambda x: x.index)

        return MultimodalEmbeddingResponse(
            output=MultimodalEmbeddingOutput(embeddings=embeddings_result),
            usage=MultimodalEmbeddingUsage(
                input_tokens=input_tokens,
                image_tokens=image_tokens,
            ),
            model=req.model,
        )

    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
