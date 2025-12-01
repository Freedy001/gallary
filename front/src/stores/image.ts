import {defineStore} from 'pinia'
import {ref, computed} from 'vue'
import {imageApi} from '@/api/image'
import type {Image, Pageable} from '@/types'

export const useImageStore = defineStore('image', () => {
  // State
  const images = ref<(Image | null)[]>([])
  const viewerIndex = ref<number>(-1);
  const selectedImages = ref<Set<number>>(new Set())
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Pagination state
  const currentPage = ref(1)
  const currentSize = ref(20)
  const total = ref(0)

  // Computed
  const selectedCount = computed(() => selectedImages.value.size)
  const selectedIds = computed(() => Array.from(selectedImages.value))

  const loadingPages = new Set<number>()
  let imageFetcher: (page: number, size: number) => Promise<Pageable<Image>>;

  // Actions
  async function fetchImages(page = 1, pageSize = 20) {
    // 防止重复加载同一页
    if (loadingPages.has(page)) return

    try {
      loadingPages.add(page)
      if (page === 1) loading.value = true
      error.value = null

      const data: Pageable<Image> = await imageFetcher(page, pageSize)

      currentPage.value = data.page
      currentSize.value = data.page_size
      total.value = data.total

      if (page === 1) {
        // 初始化数组，使用 null 占位
        const newImages = new Array(data.total).fill(null)
        data.list.forEach((item, index) => {
          newImages[index] = item
        })
        images.value = newImages
      } else {
        // 确保数组长度足够
        if (images.value.length !== data.total) {
          if (images.value.length < data.total) {
            const diff = data.total - images.value.length
            for (let i = 0; i < diff; i++) images.value.push(null)
          }
        }

        const startIndex = (page - 1) * pageSize
        data.list.forEach((item, index) => {
          if (startIndex + index < images.value.length) {
            images.value[startIndex + index] = item
          }
        })
      }

      return data
    } catch (err) {
      error.value = err instanceof Error ? err.message : '获取图片列表失败'
      throw err
    } finally {
      if (page === 1) loading.value = false
      loadingPages.delete(page)
    }
  }

  type fetchFun = (page: number, size: number) => Promise<Pageable<Image>>

  async function refreshImages(fetcher: fetchFun | null = null, pageSize = 20) {
    if (fetcher) imageFetcher = fetcher
    currentPage.value = 1
    loadingPages.clear()
    await fetchImages(1, pageSize)
  }


  async function deleteBatch(ids?: number[]) {
    try {
      const idsToDelete = ids || Array.from(selectedImages.value)
      if (idsToDelete.length === 0) return

      loading.value = true
      await imageApi.deleteBatch(idsToDelete)

      // 从列表移除已删除的图片
      images.value = images.value.filter(img => img === null || !idsToDelete.includes(img.id))
      total.value -= idsToDelete.length

      // 清除选中状态
      idsToDelete.forEach(id => selectedImages.value.delete(id))

      // 更新 storageStore 中的总数（延迟导入避免循环依赖）
      const { useStorageStore } = await import('@/stores/storage')
      const storageStore = useStorageStore()
      storageStore.updateTotalImages(-idsToDelete.length)
    } catch (err) {
      error.value = err instanceof Error ? err.message : '批量删除图片失败'
      throw err
    } finally {
      loading.value = false
    }
  }

  function selectImage(id: number) {
    selectedImages.value.add(id)
  }

  function toggleSelect(id: number) {
    if (selectedImages.value.has(id)) {
      selectedImages.value.delete(id)
    } else {
      selectedImages.value.add(id)
    }
  }

  function clearSelection() {
    selectedImages.value.clear()
  }

  return {
    // State
    images,
    viewerIndex,
    selectedImages,
    loading,
    error,
    currentPage,
    total,
    // Computed
    selectedCount,
    selectedIds,

    // Actions
    fetchImages,
    refreshImages,
    deleteBatch,
    selectImage,
    toggleSelect,
    clearSelection,
  }
})
