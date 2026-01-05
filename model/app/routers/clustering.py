"""
HDBSCAN 聚类接口
用于智能相册生成
"""

import asyncio
import uuid
from typing import List, Optional, Dict

import httpx
import numpy as np
from fastapi import APIRouter, HTTPException, BackgroundTasks
from pydantic import BaseModel, Field

try:
    import hdbscan
except ImportError:
    raise HTTPException(status_code=500, detail="缺少依赖: hdbscan，请运行 pip install hdbscan")

router = APIRouter(tags=["clustering"])

# 内存任务存储
task_store: Dict[str, Dict] = {}


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


class ClusteringRequest(BaseModel):
    """聚类请求"""
    embeddings: List[List[float]] = Field(..., description="向量列表 (N x D)")
    image_ids: List[int] = Field(..., description="对应的图片 ID 列表")
    hdbscan_params: HDBSCANParams = Field(default_factory=HDBSCANParams)
    umap_params: UMAPParams = Field(default_factory=UMAPParams)


class ClusterResult(BaseModel):
    """单个聚类结果"""
    cluster_id: int = Field(..., description="聚类 ID")
    image_ids: List[int] = Field(..., description="属于该聚类的图片 ID 列表")
    avg_probability: float = Field(..., description="该聚类的平均概率（0-1，越高表示聚类越可靠）")


class ClusteringResponse(BaseModel):
    """聚类响应"""
    clusters: List[ClusterResult] = Field(..., description="聚类结果列表")
    noise_image_ids: List[int] = Field(..., description="噪声点图片 ID 列表")
    n_clusters: int = Field(..., description="聚类数量（不含噪声）")
    params_used: dict = Field(..., description="实际使用的参数")


@router.post("/clustering", response_model=ClusteringResponse)
async def perform_clustering(request: ClusteringRequest) -> ClusteringResponse:
    """
    执行 HDBSCAN 聚类

    输入向量列表和对应的图片 ID，返回聚类结果。
    可选启用 UMAP 降维以提高高维向量的聚类效果。
    """

    embeddings = np.array(request.embeddings, dtype=np.float32)
    image_ids = request.image_ids

    if len(embeddings) != len(image_ids):
        raise HTTPException(status_code=400, detail="embeddings 和 image_ids 长度不一致")

    if len(embeddings) < 2:
        raise HTTPException(status_code=400, detail="至少需要 2 个样本进行聚类")

    # 可选 UMAP 降维
    umap_actually_used = False
    if request.umap_params.enabled and len(embeddings) > request.umap_params.n_components:
        try:
            import umap
            reducer = umap.UMAP(
                n_components=min(request.umap_params.n_components, embeddings.shape[1]),
                n_neighbors=min(request.umap_params.n_neighbors, len(embeddings) - 1),
                min_dist=request.umap_params.min_dist,
                metric="cosine",
                random_state=42
            )
            embeddings = reducer.fit_transform(embeddings)
            umap_actually_used = True
        except ImportError:
            # UMAP 不可用，继续使用原始向量
            pass

    # HDBSCAN 聚类
    params = request.hdbscan_params
    min_cluster_size = min(params.min_cluster_size, len(embeddings) // 2)
    min_cluster_size = max(min_cluster_size, 2)  # 至少为 2

    min_samples = params.min_samples
    if min_samples is not None:
        min_samples = min(min_samples, len(embeddings) - 1)

    # UMAP 降维后使用欧氏距离更合适
    metric = "euclidean" if umap_actually_used else params.metric

    clusterer = hdbscan.HDBSCAN(
        min_cluster_size=min_cluster_size,
        min_samples=min_samples,
        cluster_selection_epsilon=params.cluster_selection_epsilon,
        cluster_selection_method=params.cluster_selection_method,
        metric=metric
    )

    labels = clusterer.fit_predict(embeddings)
    probabilities = clusterer.probabilities_  # 每个样本属于其聚类的概率

    # 整理结果
    clusters_dict: dict[int, list[tuple[int, float]]] = {}  # cluster_id -> [(image_id, probability), ...]
    noise_ids: list[int] = []

    for idx, label in enumerate(labels):
        img_id = image_ids[idx]
        prob = float(probabilities[idx])
        if label == -1:
            noise_ids.append(img_id)
        else:
            if label not in clusters_dict:
                clusters_dict[label] = []
            clusters_dict[label].append((img_id, prob))

    # 构建响应，按聚类大小降序排序
    cluster_results = []
    for cluster_id, items in clusters_dict.items():
        img_ids = [item[0] for item in items]
        probs = [item[1] for item in items]
        avg_prob = sum(probs) / len(probs) if probs else 0.0
        cluster_results.append(ClusterResult(
            cluster_id=int(cluster_id),
            image_ids=img_ids,
            avg_probability=round(avg_prob, 4)
        ))

    cluster_results.sort(key=lambda x: len(x.image_ids), reverse=True)

    return ClusteringResponse(
        clusters=cluster_results,
        noise_image_ids=noise_ids,
        n_clusters=len(cluster_results),
        params_used={
            "hdbscan": {
                "min_cluster_size": min_cluster_size,
                "min_samples": min_samples,
                "cluster_selection_epsilon": params.cluster_selection_epsilon,
                "cluster_selection_method": params.cluster_selection_method,
                "metric": metric
            },
            "umap": request.umap_params.model_dump() if umap_actually_used else None
        }
    )


# ============== 异步接口 ==============

class ClusteringSubmitRequest(BaseModel):
    """异步聚类提交请求"""
    embeddings: List[List[float]] = Field(..., description="向量列表 (N x D)")
    image_ids: List[int] = Field(..., description="对应的图片 ID 列表")
    hdbscan_params: HDBSCANParams = Field(default_factory=HDBSCANParams)
    umap_params: UMAPParams = Field(default_factory=UMAPParams)
    callback_url: str = Field(..., description="Go 服务回调地址")
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


@router.post("/clustering/submit", response_model=TaskSubmitResponse)
async def submit_clustering(
    request: ClusteringSubmitRequest,
    background_tasks: BackgroundTasks
) -> TaskSubmitResponse:
    """
    提交异步聚类任务

    立即返回任务 ID，后台执行聚类，并通过回调通知 Go 服务进度
    """
    task_id = str(uuid.uuid4())
    task_store[task_id] = {
        "status": "pending",
        "progress": 0,
        "message": "任务已提交",
        "go_task_id": request.go_task_id,
        "callback_url": request.callback_url,
        "result": None,
        "error": None
    }

    # 启动后台任务
    background_tasks.add_task(run_clustering_async, task_id, request)

    return TaskSubmitResponse(task_id=task_id, status="pending")


@router.get("/clustering/status/{task_id}", response_model=TaskStatusResponse)
async def get_clustering_status(task_id: str) -> TaskStatusResponse:
    """查询任务状态（备用接口）"""
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
    try:
        # 更新进度：开始聚类
        await report_progress(task_id, "clustering", 10, "开始聚类计算")

        embeddings = np.array(request.embeddings, dtype=np.float32)
        image_ids = request.image_ids

        if len(embeddings) != len(image_ids):
            raise ValueError("embeddings 和 image_ids 长度不一致")

        if len(embeddings) < 2:
            raise ValueError("至少需要 2 个样本进行聚类")

        # 可选 UMAP 降维
        umap_actually_used = False
        if request.umap_params.enabled and len(embeddings) > request.umap_params.n_components:
            await report_progress(task_id, "clustering", 30, "UMAP 降维中")
            try:
                import umap
                reducer = umap.UMAP(
                    n_components=min(request.umap_params.n_components, embeddings.shape[1]),
                    n_neighbors=min(request.umap_params.n_neighbors, len(embeddings) - 1),
                    min_dist=request.umap_params.min_dist,
                    metric="cosine",
                    random_state=42
                )
                # 在执行器中运行 CPU 密集型任务
                loop = asyncio.get_event_loop()
                embeddings = await loop.run_in_executor(None, reducer.fit_transform, embeddings)
                umap_actually_used = True
            except ImportError:
                pass

        # HDBSCAN 聚类
        await report_progress(task_id, "clustering", 60, "HDBSCAN 聚类中")

        params = request.hdbscan_params
        min_cluster_size = min(params.min_cluster_size, len(embeddings) // 2)
        min_cluster_size = max(min_cluster_size, 2)

        min_samples = params.min_samples
        if min_samples is not None:
            min_samples = min(min_samples, len(embeddings) - 1)

        metric = "euclidean" if umap_actually_used else params.metric

        clusterer = hdbscan.HDBSCAN(
            min_cluster_size=min_cluster_size,
            min_samples=min_samples,
            cluster_selection_epsilon=params.cluster_selection_epsilon,
            cluster_selection_method=params.cluster_selection_method,
            metric=metric
        )

        # 在执行器中运行 CPU 密集型任务
        loop = asyncio.get_event_loop()
        labels = await loop.run_in_executor(None, clusterer.fit_predict, embeddings)
        probabilities = clusterer.probabilities_

        # 整理结果
        await report_progress(task_id, "clustering", 90, "整理聚类结果")

        clusters_dict: dict[int, list[tuple[int, float]]] = {}
        noise_ids: list[int] = []

        for idx, label in enumerate(labels):
            img_id = image_ids[idx]
            prob = float(probabilities[idx])
            if label == -1:
                noise_ids.append(img_id)
            else:
                if label not in clusters_dict:
                    clusters_dict[label] = []
                clusters_dict[label].append((img_id, prob))

        cluster_results = []
        for cluster_id, items in clusters_dict.items():
            img_ids = [item[0] for item in items]
            probs = [item[1] for item in items]
            avg_prob = sum(probs) / len(probs) if probs else 0.0
            cluster_results.append(ClusterResult(
                cluster_id=int(cluster_id),
                image_ids=img_ids,
                avg_probability=round(avg_prob, 4)
            ))

        cluster_results.sort(key=lambda x: len(x.image_ids), reverse=True)

        result = ClusteringResponse(
            clusters=cluster_results,
            noise_image_ids=noise_ids,
            n_clusters=len(cluster_results),
            params_used={
                "hdbscan": {
                    "min_cluster_size": min_cluster_size,
                    "min_samples": min_samples,
                    "cluster_selection_epsilon": params.cluster_selection_epsilon,
                    "cluster_selection_method": params.cluster_selection_method,
                    "metric": metric
                },
                "umap": request.umap_params.model_dump() if umap_actually_used else None
            }
        )

        # 完成，发送最终回调
        await report_completion(task_id, result)

    except Exception as e:
        await report_failure(task_id, str(e))


async def report_progress(task_id: str, status: str, progress: int, message: str):
    """主动回调 Go 服务汇报进度"""
    task = task_store.get(task_id)
    if not task:
        return

    task["status"] = status
    task["progress"] = progress
    task["message"] = message

    callback_url = task["callback_url"]
    go_task_id = task["go_task_id"]

    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            await client.post(callback_url, json={
                "python_task_id": task_id,
                "go_task_id": go_task_id,
                "status": status,
                "progress": progress,
                "message": message
            })
    except Exception as e:
        print(f"回调失败: {e}")


async def report_completion(task_id: str, result: ClusteringResponse):
    """汇报任务完成"""
    task = task_store.get(task_id)
    if not task:
        return

    task["status"] = "completed"
    task["progress"] = 100
    task["message"] = "聚类完成"
    task["result"] = result

    callback_url = task["callback_url"]
    go_task_id = task["go_task_id"]

    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            await client.post(callback_url, json={
                "python_task_id": task_id,
                "go_task_id": go_task_id,
                "status": "completed",
                "progress": 100,
                "message": "聚类完成",
                "result": result.model_dump()
            })
    except Exception as e:
        print(f"回调失败: {e}")


async def report_failure(task_id: str, error: str):
    """汇报任务失败"""
    task = task_store.get(task_id)
    if not task:
        return

    task["status"] = "failed"
    task["error"] = error
    task["message"] = f"聚类失败: {error}"

    callback_url = task["callback_url"]
    go_task_id = task["go_task_id"]

    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            await client.post(callback_url, json={
                "python_task_id": task_id,
                "go_task_id": go_task_id,
                "status": "failed",
                "progress": 0,
                "message": task["message"],
                "error": error
            })
    except Exception as e:
        print(f"回调失败: {e}")
