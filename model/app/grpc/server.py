"""
gRPC AI 服务实现
包含所有 AI 服务：嵌入、美学评分、多模态嵌入、聚类
"""

import asyncio
import io
import json
import time
from concurrent import futures
from functools import wraps

import grpc
from PIL import Image
from loguru import logger

from . import ai_pb2
from . import ai_pb2_grpc
from ..services.clustering_service import (
    clustering_service,
    HDBSCANParams,
    UMAPParams,
    ProgressInfo,
)
from ..services.embedding_service import model_service


# 配置日志
# logger = logging.getLogger(__name__)
# logging.basicConfig(
#     level=logging.INFO,
#     format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
# )


def log_grpc_request(method_name: str):
    """gRPC 请求日志装饰器"""

    def decorator(func):
        @wraps(func)
        def wrapper(self, request, context):
            start_time = time.time()
            client_ip = context.peer()

            # 记录请求开始
            logger.info(f"[{method_name}] 请求开始 - 客户端: {client_ip}")

            try:
                # 执行原方法
                response = func(self, request, context)

                # 计算处理时间
                elapsed_time = time.time() - start_time

                # 记录请求成功
                logger.info(
                    f"[{method_name}] 请求成功 - "
                    f"客户端: {client_ip}, "
                    f"处理时间: {elapsed_time:.3f}s"
                )

                return response
            except Exception as e:
                # 计算处理时间
                elapsed_time = time.time() - start_time

                # 记录请求失败
                logger.error(
                    f"[{method_name}] 请求失败 - "
                    f"客户端: {client_ip}, "
                    f"处理时间: {elapsed_time:.3f}s, "
                    f"错误: {str(e)}"
                )
                raise

        return wrapper

    return decorator


def log_grpc_stream(method_name: str):
    """gRPC 流式请求日志装饰器"""

    def decorator(func):
        @wraps(func)
        def wrapper(self, request, context):
            start_time = time.time()
            client_ip = context.peer()

            # 记录请求开始
            logger.info(f"[{method_name}] 流式请求开始 - 客户端: {client_ip}")

            try:
                # 执行原方法并返回生成器
                for response in func(self, request, context):
                    yield response

                # 计算处理时间
                elapsed_time = time.time() - start_time

                # 记录请求成功
                logger.info(
                    f"[{method_name}] 流式请求完成 - "
                    f"客户端: {client_ip}, "
                    f"总处理时间: {elapsed_time:.3f}s"
                )
            except Exception as e:
                # 计算处理时间
                elapsed_time = time.time() - start_time

                # 记录请求失败
                logger.error(
                    f"[{method_name}] 流式请求失败 - "
                    f"客户端: {client_ip}, "
                    f"处理时间: {elapsed_time:.3f}s, "
                    f"错误: {str(e)}"
                )
                raise

        return wrapper

    return decorator


# 导入评分等级函数
try:
    from train.model import get_score_level
except ImportError:
    def get_score_level(score: float) -> str:
        if score >= 9:
            return "masterpiece"
        elif score >= 8:
            return "excellent"
        elif score >= 7:
            return "very_good"
        elif score >= 6:
            return "good"
        elif score >= 5:
            return "average"
        elif score >= 4:
            return "below_average"
        elif score >= 3:
            return "poor"
        else:
            return "bad"


def load_image_from_bytes(image_bytes: bytes) -> Image.Image:
    """从二进制数据加载图片"""
    return Image.open(io.BytesIO(image_bytes)).convert("RGB")


class AIServicer(ai_pb2_grpc.AIServiceServicer):
    """gRPC AI 服务实现"""

    @log_grpc_request("Health")
    def Health(self, request, context):
        """健康检查"""
        return ai_pb2.HealthResponse(
            status="ok",
            model_loaded=model_service.is_loaded,
            device=model_service.device or "not initialized",
            backend=model_service.backend.backend_type if model_service.backend else "not initialized",
        )

    @log_grpc_request("CreateEmbedding")
    def CreateEmbedding(self, request, context):
        """创建图片嵌入向量"""
        if not model_service.is_loaded:
            context.set_code(grpc.StatusCode.UNAVAILABLE)
            context.set_details("Model not loaded")
            return ai_pb2.EmbeddingResponse()

        try:
            image_bytes_list = list(request.images)
            if not image_bytes_list:
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("Images cannot be empty")
                return ai_pb2.EmbeddingResponse()

            # 从二进制数据加载图片
            images = [load_image_from_bytes(img_bytes) for img_bytes in image_bytes_list]

            # 批量推理
            results = model_service.infer_batch(images)

            # 构建响应
            data = [
                ai_pb2.EmbeddingData(
                    index=i,
                    embedding=result.embedding.tolist(),
                )
                for i, result in enumerate(results)
            ]

            return ai_pb2.EmbeddingResponse(
                data=data,
                model=request.model or "siglip2-so400m-patch16-512",
                prompt_tokens=len(images),
                total_tokens=len(images),
            )

        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return ai_pb2.EmbeddingResponse()

    @log_grpc_request("EvaluateAesthetic")
    def EvaluateAesthetic(self, request, context):
        """评估图片美学质量"""
        if not model_service.is_loaded:
            context.set_code(grpc.StatusCode.UNAVAILABLE)
            context.set_details("Model not loaded")
            return ai_pb2.AestheticResponse()

        try:
            image_bytes_list = list(request.images)
            if not image_bytes_list:
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("Images cannot be empty")
                return ai_pb2.AestheticResponse()

            # 从二进制数据加载图片
            images = [load_image_from_bytes(img_bytes) for img_bytes in image_bytes_list]

            # 批量推理
            results = model_service.infer_batch(images)

            # 构建响应
            data = []
            for i, result in enumerate(results):
                item = ai_pb2.AestheticData(
                    index=i,
                    score=round(result.score, 2),
                    level=get_score_level(result.score),
                )
                if request.return_distribution:
                    item.distribution.extend(result.distribution.tolist())
                data.append(item)

            return ai_pb2.AestheticResponse(
                data=data,
                model="siglip2-aesthetic-lora",
                backend=model_service.backend.backend_type if model_service.backend else "unknown",
            )

        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return ai_pb2.AestheticResponse()

    @log_grpc_request("CreateMultimodalEmbedding")
    def CreateMultimodalEmbedding(self, request, context):
        """创建多模态嵌入向量"""
        if not model_service.is_loaded:
            context.set_code(grpc.StatusCode.UNAVAILABLE)
            context.set_details("Model not loaded")
            return ai_pb2.MultimodalEmbeddingResponse()

        try:
            contents = list(request.contents)
            if not contents:
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("Contents cannot be empty")
                return ai_pb2.MultimodalEmbeddingResponse()

            # 分离文本和图片
            texts = []
            text_indices = []
            images = []
            image_indices = []

            for idx, content in enumerate(contents):
                if content.HasField("text"):
                    texts.append(content.text)
                    text_indices.append(idx)
                elif content.HasField("image"):
                    # 从二进制数据加载图片
                    image = load_image_from_bytes(content.image)
                    images.append(image)
                    image_indices.append(idx)

            embeddings_result = []

            # 处理文本嵌入
            if texts:
                text_embeddings = model_service.infer_text(texts)
                for text_idx, embedding in zip(text_indices, text_embeddings):
                    embeddings_result.append(
                        ai_pb2.MultimodalEmbeddingItem(
                            index=text_idx,
                            embedding=embedding.tolist(),
                            type="text",
                        )
                    )

            # 处理图片嵌入
            if images:
                image_embeddings = model_service.infer_image_embedding(images)
                for img_idx, embedding in zip(image_indices, image_embeddings):
                    embeddings_result.append(
                        ai_pb2.MultimodalEmbeddingItem(
                            index=img_idx,
                            embedding=embedding.tolist(),
                            type="image",
                        )
                    )

            # 按原始索引排序
            embeddings_result.sort(key=lambda x: x.index)

            return ai_pb2.MultimodalEmbeddingResponse(
                embeddings=embeddings_result,
                model=request.model or "siglip2-so400m-patch16-512",
                input_tokens=len(texts),
                image_tokens=len(images),
            )

        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return ai_pb2.MultimodalEmbeddingResponse()

    @log_grpc_stream("ClusterStream")
    def ClusterStream(self, request, context):
        """流式聚类"""
        task_id = request.task_id

        # 转换参数
        embeddings = [[v for v in emb.values] for emb in request.embeddings]
        image_ids = list(request.image_ids)

        hdbscan_params = HDBSCANParams(
            min_cluster_size=request.hdbscan_params.min_cluster_size or 5,
            min_samples=request.hdbscan_params.min_samples if request.hdbscan_params.HasField('min_samples') else None,
            cluster_selection_epsilon=request.hdbscan_params.cluster_selection_epsilon or 0.0,
            cluster_selection_method=request.hdbscan_params.cluster_selection_method or "eom",
            metric=request.hdbscan_params.metric or "cosine",
        )

        umap_params = UMAPParams(
            enabled=request.umap_params.enabled,
            n_components=request.umap_params.n_components or 50,
            n_neighbors=request.umap_params.n_neighbors or 15,
            min_dist=request.umap_params.min_dist or 0.1,
        )

        # 用于在同步 gRPC 方法中运行异步代码
        loop = asyncio.new_event_loop()
        progress_queue = asyncio.Queue()

        async def progress_callback(info: ProgressInfo):
            await progress_queue.put(info)

        async def run_clustering():
            try:
                result = await clustering_service.cluster(
                    embeddings=embeddings,
                    image_ids=image_ids,
                    hdbscan_params=hdbscan_params,
                    umap_params=umap_params,
                    progress_callback=progress_callback,
                )
                await progress_queue.put(("result", result))
            except Exception as e:
                await progress_queue.put(("error", str(e)))

        task = loop.create_task(run_clustering())

        try:
            while True:
                try:
                    item = loop.run_until_complete(asyncio.wait_for(progress_queue.get(), timeout=0.1))
                except asyncio.TimeoutError:
                    if task.done():
                        break
                    continue

                if isinstance(item, tuple):
                    if item[0] == "result":
                        result = item[1]
                        clusters = [
                            ai_pb2.ClusterResult(
                                cluster_id=c.cluster_id,
                                image_ids=c.image_ids,
                                avg_probability=c.avg_probability,
                            )
                            for c in result.clusters
                        ]
                        response = ai_pb2.ClusteringResponse(
                            clusters=clusters,
                            noise_image_ids=result.noise_image_ids,
                            n_clusters=result.n_clusters,
                            params_used={k: json.dumps(v) for k, v in result.params_used.items() if v is not None},
                        )
                        yield ai_pb2.ProgressUpdate(
                            task_id=task_id,
                            status="completed",
                            progress=100,
                            message="聚类完成",
                            result=response,
                        )
                        break
                    elif item[0] == "error":
                        yield ai_pb2.ProgressUpdate(
                            task_id=task_id,
                            status="failed",
                            progress=0,
                            message=f"聚类失败: {item[1]}",
                            error=item[1],
                        )
                        break
                elif isinstance(item, ProgressInfo):
                    yield ai_pb2.ProgressUpdate(
                        task_id=task_id,
                        status=item.status,
                        progress=item.progress,
                        message=item.message,
                    )
        finally:
            loop.close()


def create_grpc_server(port: int = 50051, max_workers: int = 10) -> grpc.Server:
    """
    创建 gRPC 服务器

    Args:
        port: 监听端口
        max_workers: 最大工作线程数

    Returns:
        配置好的 gRPC 服务器
    """
    max_message_length = 500 * 1024 * 1024

    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=max_workers),
        # 【关键修改】添加 options 参数
        options=[
            ('grpc.max_receive_message_length', max_message_length),
            ('grpc.max_send_message_length', max_message_length),  # 如果你需要返回大图，这个也要改
        ]
    )
    servicer = AIServicer()
    ai_pb2_grpc.add_AIServiceServicer_to_server(servicer, server)
    server.add_insecure_port(f"0.0.0.0:{port}")
    return server
