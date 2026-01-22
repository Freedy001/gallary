/**
 * 图片缓存工具
 * 通过预加载图片并缓存 blob URL，避免重复请求
 * 使用最小堆实现优先队列，确保按位置优先级加载，视觉上从左上到右下
 */

interface QueueItem {
  url: string
  priority: number  // 加载优先级，数值越小越优先
  resolve: (blobUrl: string) => void
  reject: (error: Error) => void
}

// ==================== 最小堆实现 ====================
class MinHeap<T> {
  private heap: { value: T; priority: number }[] = []

  get size(): number {
    return this.heap.length
  }

  /**
   * 插入元素，O(log n) 复杂度
   */
  insert(value: T, priority: number): void {
    this.heap.push({ value, priority })
    this.bubbleUp(this.heap.length - 1)
  }

  /**
   * 弹出优先级最小的元素，O(log n) 复杂度
   */
  shift(): T | null {
    if (this.heap.length === 0) return null
    if (this.heap.length === 1) return this.heap.pop()!.value

    const min = this.heap[0]!.value
    this.heap[0] = this.heap.pop()!
    this.bubbleDown(0)
    return min
  }

  /**
   * 查看但不移除优先级最小的元素，O(1) 复杂度
   */
  peek(): T | null {
    return this.heap.length > 0 ? this.heap[0]!.value : null
  }

  /**
   * 清空堆
   */
  clear(): void {
    this.heap = []
  }

  /**
   * 上浮操作：将新插入的元素移动到正确位置
   */
  private bubbleUp(index: number): void {
    while (index > 0) {
      const parentIndex = Math.floor((index - 1) / 2)
      if (this.heap[parentIndex]!.priority <= this.heap[index]!.priority) {
        break
      }
      this.swap(index, parentIndex)
      index = parentIndex
    }
  }

  /**
   * 下沉操作：将堆顶元素移动到正确位置
   */
  private bubbleDown(index: number): void {
    const length = this.heap.length

    while (true) {
      const leftChild = 2 * index + 1
      const rightChild = 2 * index + 2
      let smallest = index

      if (leftChild < length && this.heap[leftChild]!.priority < this.heap[smallest]!.priority) {
        smallest = leftChild
      }
      if (rightChild < length && this.heap[rightChild]!.priority < this.heap[smallest]!.priority) {
        smallest = rightChild
      }

      if (smallest === index) break

      this.swap(index, smallest)
      index = smallest
    }
  }

  /**
   * 交换两个元素
   */
  private swap(i: number, j: number): void {
    [this.heap[i], this.heap[j]] = [this.heap[j]!, this.heap[i]!]
  }
}

// ==================== Image Cache ====================
class ImageCache {
  private cache = new Map<string, string>() // url -> blob URL
  private loading = new Map<string, Promise<string>>() // url -> loading promise
  private refCount = new Map<string, number>() // url -> reference count
  private maxSize = 500 // 最大缓存数量

  // 使用最小堆实现优先队列
  private queue = new MinHeap<QueueItem>()
  private activeCount = 0
  private maxConcurrent = 6 // 最大并发数

  /**
   * 获取缓存的图片 URL，如果未缓存则返回原始 URL
   */
  get(url: string): string {
    return this.cache.get(url) || url
  }

  /**
   * 检查是否已缓存
   */
  has(url: string): boolean {
    return this.cache.has(url)
  }

  /**
   * 增加引用计数
   */
  retain(url: string): void {
    const count = this.refCount.get(url) || 0
    this.refCount.set(url, count + 1)
  }

  /**
   * 减少引用计数
   */
  release(url: string): void {
    const count = this.refCount.get(url) || 0
    if (count > 1) {
      this.refCount.set(url, count - 1)
    } else {
      this.refCount.delete(url)
    }
  }

  /**
   * 预加载并缓存图片（通过最小堆优先队列按位置排序加载）
   * @param url 图片 URL
   * @param priority 加载优先级，数值越小越优先（通常使用图片在列表中的索引）
   */
  async preload(url: string, priority?: number): Promise<string> {
    // 已缓存直接返回
    if (this.cache.has(url)) {
      return this.cache.get(url)!
    }

    // 正在加载中，返回 promise
    if (this.loading.has(url)) {
      return this.loading.get(url)!
    }

    // 如果没有提供优先级，使用最大值（最低优先级）
    const itemPriority = priority ?? Number.MAX_SAFE_INTEGER

    // 创建一个新的 Promise 并加入堆
    const loadPromise = new Promise<string>((resolve, reject) => {
      const item: QueueItem = { url, priority: itemPriority, resolve, reject }
      this.queue.insert(item, itemPriority)
    })

    this.loading.set(url, loadPromise)

    // 尝试处理队列
    this.processQueue()

    try {
      const blobUrl = await loadPromise
      this.set(url, blobUrl)
      return blobUrl
    } finally {
      this.loading.delete(url)
    }
  }

  /**
   * 批量预加载
   */
  async preloadBatch(urls: string[]): Promise<void> {
    const unloaded = urls.filter(url => !this.cache.has(url) && !this.loading.has(url))
    await Promise.all(unloaded.map((url, index) => this.preload(url, index).catch(() => { })))
  }

  /**
   * 清除所有缓存
   */
  clear(): void {
    this.cache.forEach(blobUrl => URL.revokeObjectURL(blobUrl))
    this.cache.clear()
    this.loading.clear()
    this.refCount.clear()
    this.queue.clear()
  }

  private async loadImage(url: string): Promise<string> {
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error(`Failed to load image: ${response.status}`)
    }
    const blob = await response.blob()
    return URL.createObjectURL(blob)
  }

  /**
   * 处理加载队列（总是取优先级最高的）
   */
  private processQueue(): void {
    while (this.activeCount < this.maxConcurrent && this.queue.size > 0) {
      const item = this.queue.shift()
      if (!item) break

      this.activeCount++

      this.loadImage(item.url)
        .then(blobUrl => {
          item.resolve(blobUrl)
        })
        .catch(error => {
          item.reject(error)
        })
        .finally(() => {
          this.activeCount--
          this.processQueue()
        })
    }
  }

  private set(url: string, blobUrl: string): void {
    // 超过最大缓存数量时，删除没有被引用的最早缓存
    if (this.cache.size >= this.maxSize) {
      this.evict()
    }
    this.cache.set(url, blobUrl)
  }

  /**
   * 淘汰未被引用的缓存
   */
  private evict(): void {
    // 找到没有被引用的缓存项并删除
    for (const [url, blobUrl] of this.cache) {
      if (!this.refCount.has(url) || this.refCount.get(url) === 0) {
        URL.revokeObjectURL(blobUrl)
        this.cache.delete(url)
        this.refCount.delete(url)
        return
      }
    }
    // 如果所有缓存都被引用，不删除任何内容（允许临时超过 maxSize）
  }
}

export const thumbnailCache = new ImageCache()

