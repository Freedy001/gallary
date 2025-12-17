// WebSocket 消息类型
export type MessageType =
  | 'ai_queue_status'
  | 'storage_stats'
  | 'image_count'
  | 'ping'
  | 'pong'

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
