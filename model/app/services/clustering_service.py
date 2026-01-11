"""
聚类服务层
将 HDBSCAN 聚类业务逻辑从路由层分离
"""

from dataclasses import dataclass
from typing import Callable, Optional

import numpy as np
import umap

try:
    import hdbscan
except ImportError:
    hdbscan = None


@dataclass
class HDBSCANParams:
    """HDBSCAN 参数配置"""
    min_cluster_size: int = 5
    min_samples: Optional[int] = None
    cluster_selection_epsilon: float = 0.0
    cluster_selection_method: str = "eom"


@dataclass
class UMAPParams:
    """UMAP 降维参数"""
    enabled: bool = False
    n_components: int = 50
    n_neighbors: int = 15
    min_dist: float = 0.1


@dataclass
class ClusterResult:
    """单个聚类结果"""
    cluster_id: int
    image_ids: list[int]
    avg_probability: float


@dataclass
class ClusteringResult:
    """完整聚类结果"""
    clusters: list[ClusterResult]
    noise_image_ids: list[int]
    n_clusters: int
    params_used: dict


@dataclass
class ProgressInfo:
    """进度信息"""
    status: str
    progress: int
    message: str


# 进度回调类型（同步）
ProgressCallback = Callable[[ProgressInfo], None]


class ClusteringService:
    """聚类服务"""

    def __init__(self):
        if hdbscan is None:
            raise ImportError("缺少依赖: hdbscan，请运行 pip install hdbscan")

    def cluster(
        self,
        embeddings: list[list[float]],
        image_ids: list[int],
        hdbscan_params: HDBSCANParams,
        umap_params: UMAPParams,
        progress_callback: Optional[ProgressCallback] = None,
    ) -> ClusteringResult:
        """
        执行聚类

        Args:
            embeddings: 向量列表 (N x D)
            image_ids: 对应的图片 ID 列表
            hdbscan_params: HDBSCAN 参数
            umap_params: UMAP 降维参数
            progress_callback: 可选的进度回调函数

        Returns:
            聚类结果
        """
        # 辅助函数：报告进度
        def report(status: str, progress: int, message: str):
            if progress_callback:
                progress_callback(ProgressInfo(status, progress, message))

        report("clustering", 10, "开始聚类计算")

        embeddings_arr = np.array(embeddings, dtype=np.float32)

        if len(embeddings_arr) != len(image_ids):
            raise ValueError("embeddings 和 image_ids 长度不一致")

        if len(embeddings_arr) < 2:
            raise ValueError("至少需要 2 个样本进行聚类")

        # 可选 UMAP 降维
        umap_actually_used = False
        if umap_params.enabled and len(embeddings_arr) > umap_params.n_components:
            report("clustering", 30, "UMAP 降维中")
            reducer = umap.UMAP(
                n_components=min(umap_params.n_components, embeddings_arr.shape[1]),
                n_neighbors=min(umap_params.n_neighbors, len(embeddings_arr) - 1),
                min_dist=umap_params.min_dist,
                metric="cosine",
                random_state=42
            )
            embeddings_arr = reducer.fit_transform(embeddings_arr)
            umap_actually_used = True

        # HDBSCAN 聚类
        report("clustering", 60, "HDBSCAN 聚类中")

        min_cluster_size = min(hdbscan_params.min_cluster_size, len(embeddings_arr) // 2)
        min_cluster_size = max(min_cluster_size, 2)

        min_samples = hdbscan_params.min_samples
        if min_samples is not None:
            min_samples = min(min_samples, len(embeddings_arr) - 1)

        # 使用 euclidean 距离（对于已归一化的向量，欧氏距离等价于余弦距离）
        clusterer = hdbscan.HDBSCAN(
            min_cluster_size=min_cluster_size,
            min_samples=min_samples,
            cluster_selection_epsilon=hdbscan_params.cluster_selection_epsilon,
            cluster_selection_method=hdbscan_params.cluster_selection_method,
            metric="euclidean"
        )

        labels = clusterer.fit_predict(embeddings_arr)
        probabilities = clusterer.probabilities_

        # 整理结果
        report("clustering", 90, "整理聚类结果")

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

        params_used = {
            "hdbscan": {
                "min_cluster_size": min_cluster_size,
                "min_samples": min_samples,
                "cluster_selection_epsilon": hdbscan_params.cluster_selection_epsilon,
                "cluster_selection_method": hdbscan_params.cluster_selection_method,
                "metric": "euclidean"
            },
            "umap": {
                "enabled": umap_params.enabled,
                "n_components": umap_params.n_components,
                "n_neighbors": umap_params.n_neighbors,
                "min_dist": umap_params.min_dist,
            } if umap_actually_used else None
        }

        return ClusteringResult(
            clusters=cluster_results,
            noise_image_ids=noise_ids,
            n_clusters=len(cluster_results),
            params_used=params_used
        )


# 全局单例
clustering_service = ClusteringService()
