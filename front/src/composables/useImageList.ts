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
}

export function useImageList(options: UseImageListOptions): UseImageListReturn {
  const {originFetcher: fetcher, pageSize: defaultPageSize = 20} = options
  let overwriteFetcher: Fether | undefined = undefined

  const images = ref<(Image | null)[]>([])
  const loading = ref(false)
  const total = ref(0)
  const currentPage = ref(1)
  const selectedImages = ref<Set<number>>(new Set())
  const viewerIndex = ref(-1)
  const loadingPages = new Set<number>()

  const selectedCount = computed(() => selectedImages.value.size)
  const selectedIds = computed(() => Array.from(selectedImages.value))

  async function fetchImages(page = 1, pageSize = defaultPageSize) {
    if (loadingPages.has(page)) return

    try {
      loadingPages.add(page)
      if (page === 1) loading.value = true

      // 使用 toValue 获取当前的 fetcher，支持动态更新
      const data = await (overwriteFetcher ? overwriteFetcher(page, pageSize) : fetcher(page, pageSize))

      currentPage.value = data.page
      total.value = data.total

      if (page === 1) {
        const newImages = new Array(data.total).fill(null)
        data.list.forEach((item, index) => {
          newImages[index] = item
        })
        images.value = newImages
      } else {
        if (images.value.length < data.total) {
          const diff = data.total - images.value.length
          for (let i = 0; i < diff; i++) images.value.push(null)
        }

        const startIndex = (page - 1) * pageSize
        data.list.forEach((item, index) => {
          if (startIndex + index < images.value.length) {
            images.value[startIndex + index] = item
          }
        })
      }

      return data
    } finally {
      if (page === 1) loading.value = false
      loadingPages.delete(page)
    }
  }

  async function refresh(pageSize = defaultPageSize, fetcher?: Fether) {
    currentPage.value = 1
    loadingPages.clear()
    images.value = []
    selectedImages.value.clear()
    overwriteFetcher = fetcher
    await fetchImages(1, pageSize)
  }

  function toggleSelect(id: number) {
    if (selectedImages.value.has(id)) {
      selectedImages.value.delete(id)
    } else {
      selectedImages.value.add(id)
    }
  }

  function selectAll() {
    images.value.forEach(img => {
      if (img) selectedImages.value.add(img.id)
    })
  }

  function clearSelection() {
    selectedImages.value.clear()
  }

  function removeImages(ids: number[]) {
    images.value = images.value.filter(img => img === null || !ids.includes(img.id))
    total.value -= ids.length
    ids.forEach(id => selectedImages.value.delete(id))
  }

  return {
    images: images,
    loading: loading,
    total: total,
    currentPage: currentPage,
    selectedImages: selectedImages,
    viewerIndex: viewerIndex,
    selectedCount: selectedCount,
    selectedIds: selectedIds,
    fetchImages: fetchImages,
    refresh: refresh,
    toggleSelect: toggleSelect,
    selectAll: selectAll,
    clearSelection: clearSelection,
    removeImages: removeImages
  }
}
