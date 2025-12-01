import http from './http'

// 迁移状态
export type MigrationStatus = 'pending' | 'running' | 'completed' | 'failed' | 'rolled_back' | 'cancelled'

// 迁移任务
export interface MigrationTask {
  id: number
  status: MigrationStatus
  old_base_path: string
  old_url_prefix: string
  new_base_path: string
  new_url_prefix: string
  total_files: number
  processed_files: number
  error_message?: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}


export const migrationApi = {
  // 获取当前活跃的迁移任务
  getActive: () =>
    http.get<MigrationTask | null>('/api/storage/migration/active'),

  // 获取迁移任务详情
  getById: (id: number) =>
    http.get<MigrationTask>(`/api/storage/migration/${id}`),
}
