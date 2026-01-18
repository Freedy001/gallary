import type {StorageId} from '@/api/storage'

// 迁移类型
export type MigrationType = 'original' | 'thumbnail'

// 迁移状态
export type MigrationStatus = 'pending' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled'

// 迁移筛选条件
export interface MigrationFilterConditions {
  album_ids?: number[]
  start_date?: string
  end_date?: string
  min_file_size?: number
  max_file_size?: number
}

// 创建迁移请求
export interface CreateMigrationRequest {
  migration_type: MigrationType
  source_storage_id: StorageId
  target_storage_id: StorageId
  filter: MigrationFilterConditions
  delete_source_after_migration: boolean
}

// 迁移任务
export interface StorageMigrationTask {
  id: number
  migration_type: MigrationType
  status: MigrationStatus
  source_storage_id: StorageId
  target_storage_id: StorageId
  filter_conditions?: string
  total_files: number
  processed_files: number
  failed_files: number
  delete_source_after_migration: boolean
  error_message?: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

// 迁移预览
export interface MigrationPreview {
  files_count: number
  total_size: number
  estimated_time_seconds: number
}

// 迁移进度 VO（用于 WebSocket 推送）
export interface MigrationProgressVO {
  task_id: number
  status: MigrationStatus
  migration_type: MigrationType
  source_storage_id: StorageId
  target_storage_id: StorageId
  total_files: number
  processed_files: number
  failed_files: number
  progress_percent: number
  speed: number              // 传输速度（字节/秒）
  remaining_seconds: number  // 预计剩余时间（秒）
}

// 迁移状态 VO（包含所有活跃任务）
export interface MigrationStatusVO {
  tasks: MigrationProgressVO[]
  total_running: number
  total_paused: number
}

// 迁移历史响应
export interface MigrationHistoryResponse {
  items: StorageMigrationTask[]
  total: number
  page: number
}

// 迁移文件记录（失败文件）
export interface MigrationFileRecord {
  id: number
  task_id: number
  image_id: number
  image_name: string
  thumb_url: string
  status: string
  error_msg?: string
  created_at: string
}

// 失败文件记录响应
export interface FailedFileRecordsResponse {
  items: MigrationFileRecord[]
  total: number
  page: number
}
