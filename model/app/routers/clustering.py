"""
HDBSCAN 聚类接口
用于智能相册生成
"""

import uuid
from typing import Optional

from fastapi import APIRouter, HTTPException, BackgroundTasks
from pydantic import BaseModel, Field

from ..services.clustering_service import (
    clustering_service,
    HDBSCANParams as HDBSCANParamsDTO,
    UMAPParams as UMAPParamsDTO,
    ClusteringResult,
    ProgressInfo,
)

router = APIRouter(tags=["clustering"])

# 内存任务存储
task_store: dict[str, dict] = {}


# ============== 请求/响应模型 ==============

class HDBSCANParams(BaseModel):
    """HDBSCAN 参数配置"""
    min_cluster_size: int = Field(default=5, ge=2, description="最小聚类大小")
    min_samples: Optional[int] = Field(default=None, ge=1, description="核心点最小样本数，默认等于 min_cluster_size")
    cluster_selection_epsilon: float = Field(default=0.0, ge=0.0, description="聚类选择阈值")
    cluster_selection_method: str = Field(default="eom", description="聚类选择方法: eom 或 leaf")
    metric: str = Field(default="cosine", description="距离度量: cosine, euclidean 等")


class UMAPParams(BaseModel):
    """UMAP 降维参数（可选）"""
    enabled: bool = Field(default=False, description="是否启用 UMAP 降维")
    n_components: int = Field(default=50, ge=2, description="降维后维度")
    n_neighbors: int = Field(default=15, ge=2, description="近邻数")
    min_dist: float = Field(default=0.1, ge=0.0, le=1.0, description="最小距离")


class ClusterResultModel(BaseModel):
    """单个聚类结果"""
    cluster_id: int = Field(..., description="聚类 ID")
    image_ids: list[int] = Field(..., description="属于该聚类的图片 ID 列表")
    avg_probability: float = Field(..., description="该聚类的平均概率（0-1，越高表示聚类越可靠）")


class ClusteringResponse(BaseModel):
    """聚类响应"""
    clusters: list[ClusterResultModel] = Field(..., description="聚类结果列表")
    noise_image_ids: list[int] = Field(..., description="噪声点图片 ID 列表")
    n_clusters: int = Field(..., description="聚类数量（不含噪声）")
    params_used: dict = Field(..., description="实际使用的参数")


# ============== 异步任务接口 ==============

class ClusteringSubmitRequest(BaseModel):
    """异步聚类提交请求"""
    embeddings: list[list[float]] = Field(..., description="向量列表 (N x D)")
    image_ids: list[int] = Field(..., description="对应的图片 ID 列表")
    hdbscan_params: HDBSCANParams = Field(default_factory=HDBSCANParams)
    umap_params: UMAPParams = Field(default_factory=UMAPParams)
    go_task_id: int = Field(..., description="Go 服务的任务 ID")


class TaskSubmitResponse(BaseModel):
    """任务提交响应"""
    task_id: str = Field(..., description="Python 任务 ID")
    status: str = Field(..., description="任务状态")


class TaskStatusResponse(BaseModel):
    """任务状态响应"""
    task_id: str
    status: str  # pending | clustering | completed | failed
    progress: int  # 0-100
    message: str
    result: Optional[ClusteringResponse] = None
    error: Optional[str] = None


def _convert_result(result: ClusteringResult) -> ClusteringResponse:
    """将 service 层结果转换为 API 响应"""
    return ClusteringResponse(
        clusters=[
            ClusterResultModel(
                cluster_id=c.cluster_id,
                image_ids=c.image_ids,
                avg_probability=c.avg_probability,
            )
            for c in result.clusters
        ],
        noise_image_ids=result.noise_image_ids,
        n_clusters=result.n_clusters,
        params_used=result.params_used,
    )


@router.post("/clustering/submit", response_model=TaskSubmitResponse)
async def submit_clustering(
    request: ClusteringSubmitRequest,
    background_tasks: BackgroundTasks
) -> TaskSubmitResponse:
    """
    提交异步聚类任务

    立即返回任务 ID，后台执行聚类。
    使用 GET /clustering/status/{task_id} 轮询任务状态获取结果。
    对于流式进度更新，请使用 gRPC 端点。
    """
    task_id = str(uuid.uuid4())
    task_store[task_id] = {
        "status": "pending",
        "progress": 0,
        "message": "任务已提交",
        "go_task_id": request.go_task_id,
        "result": None,
        "error": None
    }

    # 启动后台任务
    background_tasks.add_task(run_clustering_async, task_id, request)

    return TaskSubmitResponse(task_id=task_id, status="pending")


@router.get("/clustering/status/{task_id}", response_model=TaskStatusResponse)
async def get_clustering_status(task_id: str) -> TaskStatusResponse:
    """查询任务状态"""
    task = task_store.get(task_id)
    if not task:
        raise HTTPException(status_code=404, detail=f"任务 {task_id} 不存在")

    return TaskStatusResponse(
        task_id=task_id,
        status=task["status"],
        progress=task["progress"],
        message=task["message"],
        result=task.get("result"),
        error=task.get("error")
    )


async def run_clustering_async(task_id: str, request: ClusteringSubmitRequest):
    """后台异步执行聚类"""
    task = task_store.get(task_id)
    if not task:
        return

    async def progress_callback(info: ProgressInfo):
        """进度回调 - 更新内存任务状态"""
        task["status"] = info.status
        task["progress"] = info.progress
        task["message"] = info.message

    try:
        # 转换参数
        hdbscan_params = HDBSCANParamsDTO(
            min_cluster_size=request.hdbscan_params.min_cluster_size,
            min_samples=request.hdbscan_params.min_samples,
            cluster_selection_epsilon=request.hdbscan_params.cluster_selection_epsilon,
            cluster_selection_method=request.hdbscan_params.cluster_selection_method,
            metric=request.hdbscan_params.metric,
        )

        umap_params = UMAPParamsDTO(
            enabled=request.umap_params.enabled,
            n_components=request.umap_params.n_components,
            n_neighbors=request.umap_params.n_neighbors,
            min_dist=request.umap_params.min_dist,
        )

        # 执行聚类
        result = await clustering_service.cluster(
            embeddings=request.embeddings,
            image_ids=request.image_ids,
            hdbscan_params=hdbscan_params,
            umap_params=umap_params,
            progress_callback=progress_callback,
        )

        # 更新任务状态
        task["status"] = "completed"
        task["progress"] = 100
        task["message"] = "聚类完成"
        task["result"] = _convert_result(result)

    except Exception as e:
        task["status"] = "failed"
        task["error"] = str(e)
        task["message"] = f"聚类失败: {e}"
