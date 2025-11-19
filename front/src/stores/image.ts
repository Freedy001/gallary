import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { imageApi } from '@/api/image'
import type { Image, Pageable, SearchParams } from '@/types'

export const useImageStore = defineStore('image', () => {
  // State
  const images = ref<Image[]>([])
  const currentImage = ref<Image | null>(null)
  const selectedImages = ref<Set<number>>(new Set())
  const loading = ref(false)
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
    try {
      loading.value = true
      error.value = null

      const response = isSearchMode.value && searchParams.value
        ? await imageApi.search({ ...searchParams.value, page, page_size: pageSize.value })
        : await imageApi.getList(page, pageSize.value)

      const data: Pageable<Image> = response.data

      if (page === 1) {
        images.value = data.items
      } else {
        images.value.push(...data.items)
      }

      currentPage.value = data.page
      pageSize.value = data.page_size
      total.value = data.total
      totalPages.value = data.total_pages

      return data
    } catch (err) {
      error.value = err instanceof Error ? err.message : '获取图片列表失败'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function loadMore() {
    if (!hasMore.value || loading.value) return
    await fetchImages(currentPage.value + 1)
  }

  async function refreshImages() {
    currentPage.value = 1
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
      images.value = images.value.filter(img => img.id !== id)
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
    selectImage,
    deselectImage,
    toggleSelect,
    clearSelection,
    setCurrentImage,
  }
})
