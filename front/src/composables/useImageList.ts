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

    const existingIds = new Set(this.images.value.filter((img): img is Image => img !== null).map(img => img.id))
    const imagesToInsert = newImages.filter(img => !existingIds.has(img.id))
    if (imagesToInsert.length === 0) return

    const allImages = [...this.images.value, ...imagesToInsert]
    // 使用 taken_at 排序，如果没有 taken_at 则使用 created_at
    allImages.sort((a, b) => {
      const timeA = a?.taken_at ? new Date(a.taken_at).getTime() : (a?.created_at ? new Date(a.created_at).getTime() : 0)
      const timeB = b?.taken_at ? new Date(b.taken_at).getTime() : (b?.created_at ? new Date(b.created_at).getTime() : 0)
      return timeB - timeA
    })

    this.images.value = allImages
    this.total.value += imagesToInsert.length
  }
}

export function useImageList(options: UseImageListOptions): UseImageListReturn {
  return new ImageListManager(options)
}
