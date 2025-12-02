/**
 * 图片缓存工具
 * 通过预加载图片并缓存 blob URL，避免重复请求
 */

class ImageCache {
  private cache = new Map<string, string>() // url -> blob URL
  private loading = new Map<string, Promise<string>>() // url -> loading promise
  private maxSize = 200 // 最大缓存数量

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
   * 预加载并缓存图片
   */
  async preload(url: string): Promise<string> {
    // 已缓存直接返回
    if (this.cache.has(url)) {
      return this.cache.get(url)!
    }

    // 正在加载中，返回 promise
    if (this.loading.has(url)) {
      return this.loading.get(url)!
    }

    // 开始加载
    const loadPromise = this.loadImage(url)
    this.loading.set(url, loadPromise)

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
    await Promise.all(unloaded.map(url => this.preload(url).catch(() => {})))
  }

  private async loadImage(url: string): Promise<string> {
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error(`Failed to load image: ${response.status}`)
    }
    const blob = await response.blob()
    return URL.createObjectURL(blob)
  }

  private set(url: string, blobUrl: string): void {
    // 超过最大缓存数量时，删除最早的缓存
    if (this.cache.size >= this.maxSize) {
      const firstKey = this.cache.keys().next().value
      if (firstKey) {
        const oldBlobUrl = this.cache.get(firstKey)
        if (oldBlobUrl) {
          URL.revokeObjectURL(oldBlobUrl)
        }
        this.cache.delete(firstKey)
      }
    }
    this.cache.set(url, blobUrl)
  }

  /**
   * 清除所有缓存
   */
  clear(): void {
    this.cache.forEach(blobUrl => URL.revokeObjectURL(blobUrl))
    this.cache.clear()
    this.loading.clear()
  }
}

export const thumbnailCache = new ImageCache()
