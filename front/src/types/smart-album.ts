
// HDBSCAN 参数
export interface HDBSCANParams {
  min_cluster_size: number
  min_samples?: number | null
  cluster_selection_epsilon: number
  cluster_selection_method: 'eom' | 'leaf'
  // UMAP 降维参数
  umap_enabled: boolean
  umap_n_components?: number
  umap_n_neighbors?: number
  umap_min_dist?: number
}

// 智能相册配置
export interface SmartAlbumConfig {
  model_name: string
  algorithm: string
  cluster_id: number
  generated_at: string
  hdbscan_params?: HDBSCANParams
  image_count: number
  avg_probability: number
}

// 生成智能相册请求
export interface GenerateSmartAlbumsRequest {
  model_name: string
  algorithm: 'hdbscan'
  hdbscan_params: HDBSCANParams
}

// 默认 HDBSCAN 参数（推荐值）
export const DEFAULT_HDBSCAN_PARAMS: HDBSCANParams = {
  min_cluster_size: 5,
  min_samples: null,
  cluster_selection_epsilon: 0.0,
  cluster_selection_method: 'eom',
  // UMAP 默认参数
  umap_enabled: false,
  umap_n_components: 50,
  umap_n_neighbors: 15,
  umap_min_dist: 0.1
}

// 智能相册任务状态
export type SmartAlbumTaskStatus =
  | 'pending'      // 待处理
  | 'collecting'   // 收集向量
  | 'clustering'   // 聚类中
  | 'creating'     // 创建相册
  | 'completed'    // 完成
  | 'failed'       // 失败

export interface SmartAlbumProgressVO {
  task_id: number
  status: SmartAlbumTaskStatus
  progress: number
  message: string
  error?: string
  album_ids?: number[]
  cluster_count?: number
  noise_count?: number
  noise_image_ids?: number[]
  total_images?: number
}
