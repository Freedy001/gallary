# Routers module
from .aesthetics import router as aesthetics_router
from .embeddings import router as embeddings_router
from .multimodal_embedding import router as multimodal_embedding_router

__all__ = [
    "embeddings_router",
    "aesthetics_router",
    "multimodal_embedding_router",
]
