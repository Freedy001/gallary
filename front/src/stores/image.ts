import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { imageApi } from '@/api/image'
import type { Image, Pageable, SearchParams } from '@/types'

export const useImageStore = defineStore('image', () => {
  // State
  const images = ref<(Image | null)[]>([])
  const currentImage = ref<Image | null>(null)
  const selectedImages = ref<Set<number>>(new Set())
  const loading = ref(false)
  const loadingPages = ref<Set<number>>(new Set())
  const error = ref<string | null>(null)

  // Pagination state
  const currentPage = ref(1)
  const pageSize = ref(20)
  const total = ref(0)
  const totalPages = ref(0)

  // Search state
  const searchParams = ref<SearchParams | null>(null)
  const isSearchMode = ref(false)

  // Computed
  const hasMore = computed(() => currentPage.value < totalPages.value)
  const selectedCount = computed(() => selectedImages.value.size)

  // Actions
  async function fetchImages(page = 1) {
    // 防止重复加载同一页
    if (loadingPages.value.has(page)) return

    try {
      loadingPages.value.add(page)
      if (page === 1) loading.value = true
      error.value = null

      const response = isSearchMode.value && searchParams.value
        ? await imageApi.search({ ...searchParams.value, page, page_size: pageSize.value })
        : await imageApi.getList(page, pageSize.value)

      const data: Pageable<Image> = response.data

      currentPage.value = data.page
      pageSize.value = data.page_size
      total.value = data.total
      totalPages.value = data.total_pages

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

        const startIndex = (page - 1) * pageSize.value
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
      loadingPages.value.delete(page)
    }
  }

  async function loadMore() {
    if (!hasMore.value || loading.value) return
    await fetchImages(currentPage.value + 1)
  }

  async function refreshImages() {
    currentPage.value = 1
    loadingPages.value.clear()
    await fetchImages(1)
  }

  async function searchImages(params: SearchParams) {
    searchParams.value = params
    isSearchMode.value = true
    currentPage.value = 1
    await fetchImages(1)
  }

  function clearSearch() {
    searchParams.value = null
    isSearchMode.value = false
    refreshImages()
  }

  async function getImageDetail(id: number) {
    try {
      loading.value = true
      const response = await imageApi.getDetail(id)
      currentImage.value = response.data
      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : '获取图片详情失败'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteImage(id: number) {
    try {
      await imageApi.delete(id)
      images.value = images.value.filter(img => img === null || img.id !== id)
      total.value -= 1

      if (currentImage.value?.id === id) {
        currentImage.value = null
      }

      selectedImages.value.delete(id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : '删除图片失败'
      throw err
    }
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

      // 如果当前查看的图片被删除，清空当前图片
      if (currentImage.value && idsToDelete.includes(currentImage.value.id)) {
        currentImage.value = null
      }
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

  function deselectImage(id: number) {
    selectedImages.value.delete(id)
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

  function setCurrentImage(image: Image | null) {
    currentImage.value = image
  }

  return {
    // State
    images,
    currentImage,
    selectedImages,
    loading,
    error,
    currentPage,
    pageSize,
    total,
    totalPages,
    searchParams,
    isSearchMode,

    // Computed
    hasMore,
    selectedCount,

    // Actions
    fetchImages,
    loadMore,
    refreshImages,
    searchImages,
    clearSearch,
    getImageDetail,
    deleteImage,
    deleteBatch,
    selectImage,
    deselectImage,
    toggleSelect,
    clearSelection,
    setCurrentImage,
  }
})
