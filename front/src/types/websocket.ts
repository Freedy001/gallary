// WebSocket 消息类型
export type MessageType =
  | 'ai_queue_status'
  | 'storage_stats'
  | 'image_count'
  | 'smart_album_progress'
  | 'ping'
  | 'pong'
  // 数据同步事件（后端推送）
  | 'images_restored'      // 图片从回收站恢复
  | 'images_deleted'       // 图片被删除
  | 'images_uploaded'      // 图片上传完成
  | 'albums_updated'       // 相册更新（AI命名完成等）

// WebSocket 消息结构
export interface WSMessage<T = unknown> {
  type: MessageType
  data?: T
  timestamp: number
  request_id?: string
}

// 连接状态
export type ConnectionState =
  | 'disconnected'
  | 'connecting'
  | 'connected'
  | 'reconnecting'

// WebSocket 事件回调
export interface WSEventHandlers {
  onConnected?: () => void
  onDisconnected?: () => void
  onMessage?: (message: WSMessage) => void
  onError?: (error: Event) => void
  onReconnecting?: (attempt: number) => void
}
