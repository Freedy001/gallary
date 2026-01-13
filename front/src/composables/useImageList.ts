import {computed, ref, type Ref} from 'vue'
import type {Image, Pageable} from '@/types'


export type Fether = (page: number, size: number) => Promise<Pageable<Image>>

export interface UseImageListOptions {
  originFetcher: Fether
  pageSize?: number
}

export interface UseImageListReturn {
  images: Ref<(Image | null)[]>
  loading: Ref<boolean>
  total: Ref<number>
  currentPage: Ref<number>
  selectedImages: Ref<Set<number>>
  viewerIndex: Ref<number>
  selectedCount: Ref<number>
  selectedIds: Ref<number[]>
  fetchImages: (page?: number, pageSize?: number) => Promise<Pageable<Image> | undefined>
  refresh: (pageSize?: number, fetcher?: Fether) => Promise<void>
  toggleSelect: (id: number) => void
  selectAll: () => void
  clearSelection: () => void
  removeImages: (ids: number[]) => void
  insertImages: (newImages: Image[]) => void
}

export class ImageListManager implements UseImageListReturn {
  images: Ref<(Image | null)[]>
  loading: Ref<boolean>
  total: Ref<number>
  currentPage: Ref<number>
  selectedImages: Ref<Set<number>>
  viewerIndex: Ref<number>
  selectedCount: Ref<number>
  selectedIds: Ref<number[]>

  private readonly originFetcher: Fether
  private defaultPageSize: number
  private overwriteFetcher?: Fether
  private loadingPages: Set<number>

  constructor(options: UseImageListOptions) {
    const {originFetcher, pageSize = 20} = options

    this.originFetcher = originFetcher
    this.defaultPageSize = pageSize
    this.overwriteFetcher = undefined
    this.loadingPages = new Set<number>()

    this.images = ref<(Image | null)[]>([])
    this.loading = ref(false)
    this.total = ref(0)
    this.currentPage = ref(1)
    this.selectedImages = ref<Set<number>>(new Set())
    this.viewerIndex = ref(-1)

    this.selectedCount = computed(() => this.selectedImages.value.size)
    this.selectedIds = computed(() => Array.from(this.selectedImages.value))
  }

  fetchImages = async (page = 1, pageSize = this.defaultPageSize) => {
    if (this.loadingPages.has(page)) return

    try {
      this.loadingPages.add(page)
      if (page === 1) this.loading.value = true

      const data = await (this.overwriteFetcher
        ? this.overwriteFetcher(page, pageSize)
        : this.originFetcher(page, pageSize))

      this.currentPage.value = data.page || 1
      this.total.value = data.total || 0

      if (page === 1) {
        const newImages = new Array(data.total).fill(null)
        data.list.forEach((item, index) => {
          newImages[index] = item
        })
        this.images.value = newImages
      } else {
        if (this.images.value.length < data.total) {
          const diff = data.total - this.images.value.length
          for (let i = 0; i < diff; i++) this.images.value.push(null)
        }

        const startIndex = (page - 1) * pageSize
        data.list.forEach((item, index) => {
          if (startIndex + index < this.images.value.length) {
            this.images.value[startIndex + index] = item
          }
        })
      }

      return data
    } finally {
      if (page === 1) this.loading.value = false
      this.loadingPages.delete(page)
    }
  }

  refresh = async (pageSize = this.defaultPageSize, fetcher?: Fether) => {
    this.currentPage.value = 1
    this.loadingPages.clear()
    this.images.value = []
    this.selectedImages.value.clear()
    this.overwriteFetcher = fetcher
    await this.fetchImages(1, pageSize)
  }

  toggleSelect = (id: number) => {
    if (this.selectedImages.value.has(id)) {
      this.selectedImages.value.delete(id)
    } else {
      this.selectedImages.value.add(id)
    }
  }

  selectAll = () => {
    this.images.value.forEach(img => {
      if (img) this.selectedImages.value.add(img.id)
    })
  }

  clearSelection = () => {
    this.selectedImages.value.clear()
  }

  removeImages = (ids: number[]) => {
    this.images.value = this.images.value.filter(img => img === null || !ids.includes(img.id))
    this.total.value -= ids.length
    ids.forEach(id => this.selectedImages.value.delete(id))
  }

  insertImages = (newImages: Image[]) => {
    if (newImages.length === 0) return

    // 1. 过滤重复 (保持原有逻辑)
    const existingIds = new Set(this.images.value.filter((img): img is Image => img !== null).map(img => img.id))
    const imagesToInsert = newImages.filter(img => !existingIds.has(img.id))
    if (imagesToInsert.length === 0) return

    // 2. 辅助函数
    const getSortKey = (img: Image | null | undefined): number => {
      if (!img) return 0
      return img.taken_at ? new Date(img.taken_at).getTime() : (img.created_at ? new Date(img.created_at).getTime() : 0)
    }

    // 3. 对新图片排序 (降序)
    imagesToInsert.sort((a, b) => getSortKey(b) - getSortKey(a))

    // 4. 双指针合并算法 (O(M+N) 高性能)
    const oldImages = this.images.value
    const mergedImages: (Image | null)[] = []

    let i = 0 // 指向 oldImages
    let j = 0 // 指向 imagesToInsert

    while (i < oldImages.length && j < imagesToInsert.length) {
      const oldKey = getSortKey(oldImages[i])
      const newKey = getSortKey(imagesToInsert[j])

      // 降序：谁大谁先进入数组
      // 如果 key 相等，通常让新图片在前(或者在后，取决于你的需求)，这里假设新图片排在旧图片前
      if (newKey >= oldKey) {
        mergedImages.push(imagesToInsert[j] || null)
        j++
      } else {
        mergedImages.push(oldImages[i] || null)
        i++
      }
    }

    // 5. 处理剩余的元素
    if (i < oldImages.length) {
      // 剩下的旧图片追加进去
      // 性能提示：如果是 Vue 的 Ref 数组，使用 push (...items) 可能比 concat 快，视具体环境而定
      // 但最简单的写法是 concat
      mergedImages.push(...oldImages.slice(i))
    }

    if (j < imagesToInsert.length) {
      // 剩下的新图片追加进去
      mergedImages.push(...imagesToInsert.slice(j))
    }

    // 6. 一次性赋值，触发一次响应式更新
    this.images.value = mergedImages
    this.total.value += imagesToInsert.length
  }
}

export function useImageList(options: UseImageListOptions): UseImageListReturn {
  return new ImageListManager(options)
}
