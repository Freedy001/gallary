import {defineStore} from 'pinia'
import {computed, ref} from 'vue'
import {albumApi} from '@/api/album'
import type {Album} from '@/types'

export const useAlbumStore = defineStore('album', () => {
  // State
  const albums = ref<Album[]>([])
  const currentAlbum = ref<Album | null>(null)
  const loading = ref(false)
  const total = ref(0)
  const selectedAlbums = ref<Set<number>>(new Set())

  // Computed
  const selectedCount = computed(() => selectedAlbums.value.size)
  const hasSelection = computed(() => selectedAlbums.value.size > 0)
  const selectedAlbumList = computed(() =>
    albums.value.filter(a => selectedAlbums.value.has(a.id))
  )

  // Actions
  async function fetchAlbums(page = 1, pageSize = 20) {
    try {
      loading.value = true
      const { data } = await albumApi.getList({ page, pageSize })
      albums.value = data.list
      total.value = data.total
    } finally {
      loading.value = false
    }
  }

  async function createAlbum(name: string, description?: string) {
    const { data } = await albumApi.create({ name, description })
    albums.value.unshift(data)
    total.value += 1
    return data
  }

  async function updateAlbum(id: number, name: string, description?: string) {
    const { data } = await albumApi.update(id, { name, description })
    const index = albums.value.findIndex(a => a.id === id)
    if (index !== -1) {
      albums.value[index] = { ...albums.value[index], ...data }
    }
    if (currentAlbum.value?.id === id) {
      currentAlbum.value = { ...currentAlbum.value, ...data }
    }
    return data
  }

  async function setAlbumCover(id: number, imageId: number) {
    await albumApi.setCover(id, imageId)
  }

  async function deleteAlbum(id: number) {
    await albumApi.delete(id)
    albums.value = albums.value.filter(a => a.id !== id)
    total.value -= 1
    selectedAlbums.value.delete(id)
    if (currentAlbum.value?.id === id) {
      currentAlbum.value = null
    }
  }

  async function deleteSelectedAlbums() {
    const ids = Array.from(selectedAlbums.value)
    for (const id of ids) {
      await albumApi.delete(id)
    }
    albums.value = albums.value.filter(a => !selectedAlbums.value.has(a.id))
    total.value -= ids.length
    if (currentAlbum.value && selectedAlbums.value.has(currentAlbum.value.id)) {
      currentAlbum.value = null
    }
    selectedAlbums.value.clear()
  }

  function toggleAlbumSelection(id: number) {
    if (selectedAlbums.value.has(id)) {
      selectedAlbums.value.delete(id)
    } else {
      selectedAlbums.value.add(id)
    }
  }

  function clearSelection() {
    selectedAlbums.value.clear()
  }

  function selectAll() {
    albums.value.forEach(a => selectedAlbums.value.add(a.id))
  }

  function clearCurrentAlbum() {
    currentAlbum.value = null
  }

  return {
    // State
    albums,
    currentAlbum,
    loading,
    total,
    selectedAlbums,
    // Computed
    selectedCount,
    hasSelection,
    selectedAlbumList,
    // Actions
    fetchAlbums,
    createAlbum,
    updateAlbum,
    setAlbumCover,
    deleteAlbum,
    deleteSelectedAlbums,
    toggleAlbumSelection,
    clearSelection,
    selectAll,
    clearCurrentAlbum,
  }
})
