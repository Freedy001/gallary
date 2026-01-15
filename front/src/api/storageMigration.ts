import http from './http'
import type {
  CreateMigrationRequest,
  FailedFileRecordsResponse,
  MigrationPreview,
  MigrationStatusVO,
  StorageMigrationTask,
} from '@/types/migration'

export const storageMigrationApi = {
  // 创建迁移任务
  createMigration: (req: CreateMigrationRequest) =>
    http.post<StorageMigrationTask>('/api/storage/storage-migration', req),

  // 预览迁移
  previewMigration: (req: CreateMigrationRequest) =>
    http.post<MigrationPreview>('/api/storage/storage-migration/preview', req),

  // 获取所有活跃迁移任务列表
  listActiveMigrations: () =>
    http.get<MigrationStatusVO>('/api/storage/storage-migration/list/active'),

  // 暂停迁移任务
  pauseMigration: (id: number) =>
    http.post<{ message: string }>(`/api/storage/storage-migration/${id}/pause`),

  // 恢复迁移任务
  resumeMigration: (id: number) =>
    http.post<{ message: string }>(`/api/storage/storage-migration/${id}/resume`),

  // 获取失败文件记录
  getFailedFileRecords: (id: number, page = 1, pageSize = 20) =>
    http.get<FailedFileRecordsResponse>(`/api/storage/storage-migration/${id}/failed`, {
      params: { page, page_size: pageSize },
    }),

  // 重试失败文件
  retryFailedFiles: (id: number) =>
    http.post<{ message: string }>(`/api/storage/storage-migration/${id}/retry`),

  // 忽略失败文件
  dismissFailedFiles: (id: number) =>
    http.post<{ message: string }>(`/api/storage/storage-migration/${id}/dismiss`),
}
