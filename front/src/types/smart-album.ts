import type {Album} from './album'

// HDBSCAN 参数
export interface HDBSCANParams {
  min_cluster_size: number
  min_samples?: number | null
  cluster_selection_epsilon: number
  cluster_selection_method: 'eom' | 'leaf'
  metric: 'cosine' | 'euclidean'
  umap_enabled: boolean
  umap_n_components?: number
  umap_n_neighbors?: number
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

// 生成智能相册响应
export interface GenerateSmartAlbumsResponse {
  albums: Album[]
  noise_count: number
  total_images: number
}

// 默认 HDBSCAN 参数（推荐值）
export const DEFAULT_HDBSCAN_PARAMS: HDBSCANParams = {
  min_cluster_size: 5,
  min_samples: null,
  cluster_selection_epsilon: 0.0,
  cluster_selection_method: 'eom',
  metric: 'cosine',
  umap_enabled: false,
  umap_n_components: 50,
  umap_n_neighbors: 15
}

// 智能相册任务状态
export type SmartAlbumTaskStatus =
  | 'pending'      // 待处理
  | 'collecting'   // 收集向量
  | 'clustering'   // 聚类中
  | 'creating'     // 创建相册
  | 'completed'    // 完成
  | 'failed'       // 失败

// 智能相册任务扩展信息
export interface SmartAlbumTaskExtra {
  progress: number
  message: string
  python_task_id?: string
  model_name: string
  algorithm: string
  hdbscan_params?: HDBSCANParams
  album_ids?: number[]
  cluster_count?: number
  noise_count?: number
  total_images?: number
}

// 智能相册任务 VO
export interface SmartAlbumTaskVO {
  id: number
  status: SmartAlbumTaskStatus
  progress: number
  message: string
  config?: SmartAlbumTaskExtra
  result?: SmartAlbumTaskExtra
  error?: string
  created_at: string
  updated_at: string
}
