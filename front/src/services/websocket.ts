import type {ConnectionState, MessageType, WSEventHandlers, WSMessage} from '@/types/websocket'

class WebSocketService {
  private ws: WebSocket | null = null
  private url: string = ''
  private state: ConnectionState = 'disconnected'

  // 重连配置
  private reconnectAttempt = 0
  private maxReconnectDelay = 30000  // 最大 30 秒
  private baseDelay = 1000           // 基础 1 秒
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null

  // 心跳配置
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null
  private heartbeatTimeout = 30000   // 30 秒发送一次心跳
  private lastPongTime = 0

  // 事件处理器
  private handlers: WSEventHandlers = {}

  // 消息订阅者
  private subscribers: Map<MessageType, Set<(data: unknown) => void>> = new Map()

  /**
   * 初始化 WebSocket 连接
   */
  connect(handlers?: WSEventHandlers): void {
    if (this.state === 'connected' || this.state === 'connecting') {
      return
    }

    this.handlers = handlers || {}
    this.state = 'connecting'

    // 构建 WebSocket URL
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const token = localStorage.getItem('auth_token') || ''
    this.url = `${protocol}//${host}/api/ws?token=${encodeURIComponent(token)}`

    this.createConnection()
  }

  /**
   * 创建 WebSocket 连接
   */
  private createConnection(): void {
    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        console.log('[WebSocket] 连接成功')
        this.state = 'connected'
        this.reconnectAttempt = 0
        this.startHeartbeat()
        this.handlers.onConnected?.()
      }

      this.ws.onclose = (event) => {
        console.log('[WebSocket] 连接关闭', event.code, event.reason)
        this.stopHeartbeat()

        if (this.state !== 'disconnected') {
          this.state = 'reconnecting'
          this.scheduleReconnect()
        }

        this.handlers.onDisconnected?.()
      }

      this.ws.onerror = (error) => {
        console.error('[WebSocket] 连接错误', error)
        this.handlers.onError?.(error)
      }

      this.ws.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data)
          this.handleMessage(message)
        } catch (e) {
          console.error('[WebSocket] 消息解析失败', e)
        }
      }
    } catch (error) {
      console.error('[WebSocket] 创建连接失败', error)
      this.scheduleReconnect()
    }
  }

  /**
   * 处理接收到的消息
   */
  private handleMessage(message: WSMessage): void {
    // 处理心跳响应
    if (message.type === 'pong') {
      this.lastPongTime = Date.now()
      return
    }

    // 通知全局处理器
    this.handlers.onMessage?.(message)

    // 通知特定类型的订阅者
    const subscribers = this.subscribers.get(message.type)
    if (subscribers) {
      subscribers.forEach(callback => {
        try {
          callback(message.data)
        } catch (e) {
          console.error('[WebSocket] 订阅回调执行失败', e)
        }
      })
    }
  }

  /**
   * 启动心跳
   */
  private startHeartbeat(): void {
    this.stopHeartbeat()
    this.lastPongTime = Date.now()

    this.heartbeatInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        // 检查上次 pong 响应时间
        if (Date.now() - this.lastPongTime > this.heartbeatTimeout * 2) {
          console.warn('[WebSocket] 心跳超时，重新连接')
          this.ws.close()
          return
        }

        this.send({ type: 'ping', timestamp: Date.now() })
      }
    }, this.heartbeatTimeout)
  }

  /**
   * 停止心跳
   */
  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval)
      this.heartbeatInterval = null
    }
  }

  /**
   * 指数退避重连
   */
  private scheduleReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
    }

    this.reconnectAttempt++

    // 指数退避：1s, 2s, 4s, 8s, 16s, 30s（最大）
    const delay = Math.min(
      this.baseDelay * Math.pow(2, this.reconnectAttempt - 1),
      this.maxReconnectDelay
    )

    console.log(`[WebSocket] 将在 ${delay}ms 后重连 (第 ${this.reconnectAttempt} 次)`)
    this.handlers.onReconnecting?.(this.reconnectAttempt)

    this.reconnectTimer = setTimeout(() => {
      this.createConnection()
    }, delay)
  }

  /**
   * 订阅特定消息类型
   */
  subscribe<T>(type: MessageType, callback: (data: T) => void): () => void {
    if (!this.subscribers.has(type)) {
      this.subscribers.set(type, new Set())
    }

    const callbacks = this.subscribers.get(type)!
    callbacks.add(callback as (data: unknown) => void)

    // 返回取消订阅函数
    return () => {
      callbacks.delete(callback as (data: unknown) => void)
      if (callbacks.size === 0) {
        this.subscribers.delete(type)
      }
    }
  }

  /**
   * 发送消息
   */
  send(message: Partial<WSMessage>): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        ...message,
        timestamp: message.timestamp || Date.now()
      }))
    }
  }

  /**
   * 断开连接
   */
  disconnect(): void {
    this.state = 'disconnected'
    this.stopHeartbeat()

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }

    if (this.ws) {
      this.ws.close()
      this.ws = null
    }

    this.subscribers.clear()
  }

  /**
   * 获取连接状态
   */
  getState(): ConnectionState {
    return this.state
  }

  /**
   * 是否已连接
   */
  isConnected(): boolean {
    return this.state === 'connected'
  }
}

// 导出单例
export const wsService = new WebSocketService()
export default wsService
