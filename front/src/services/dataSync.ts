/**
 * 数据同步事件服务
 *
 * 用于在不同组件/页面之间同步数据变更，解决 keep-alive 缓存导致的数据不同步问题。
 * 支持本地事件触发和 WebSocket 远程事件转发。
 */

// 数据同步事件类型
export type DataSyncEvent =
  | 'images:restored'           // 图片从回收站恢复
  | 'images:deleted'            // 图片被删除(移到回收站)
  | 'images:permanentlyDeleted' // 图片彻底删除
  | 'images:uploaded'           // 图片上传完成
  | 'albums:updated'            // 相册更新(名称/封面等)
  | 'albums:created'            // 相册创建
  | 'albums:deleted'            // 相册删除

// 事件载荷
export interface DataSyncPayload {
  ids?: number[]        // 图片 ID 列表
  albumIds?: number[]   // 相册 ID 列表
  source?: string       // 来源标识，避免重复刷新
}

type EventCallback = (payload: DataSyncPayload) => void

class DataSyncService {
  private listeners: Map<DataSyncEvent, Set<EventCallback>> = new Map()

  /**
   * 发送数据同步事件
   */
  emit(event: DataSyncEvent, payload: DataSyncPayload = {}): void {
    const callbacks = this.listeners.get(event)
    if (callbacks) {
      callbacks.forEach(callback => {
        try {
          callback(payload)
        } catch (e) {
          console.error(`[DataSync] 事件处理失败: ${event}`, e)
        }
      })
    }
  }

  /**
   * 监听数据同步事件
   * @returns 取消订阅函数
   */
  on(event: DataSyncEvent, callback: EventCallback): () => void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }

    const callbacks = this.listeners.get(event)!
    callbacks.add(callback)

    // 返回取消订阅函数
    return () => {
      callbacks.delete(callback)
      if (callbacks.size === 0) {
        this.listeners.delete(event)
      }
    }
  }

  /**
   * 取消监听
   */
  off(event: DataSyncEvent, callback: EventCallback): void {
    const callbacks = this.listeners.get(event)
    if (callbacks) {
      callbacks.delete(callback)
      if (callbacks.size === 0) {
        this.listeners.delete(event)
      }
    }
  }

  /**
   * 清除所有监听器
   */
  clear(): void {
    this.listeners.clear()
  }
}

// 导出单例
export const dataSyncService = new DataSyncService()
