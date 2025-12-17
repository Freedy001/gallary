import { defineStore } from 'pinia'
import { ref } from 'vue'
import { albumApi } from '@/api/album'
import type { Album } from '@/types'

export const useAlbumStore = defineStore('album', () => {
  // State
  const albums = ref<Album[]>([])
  const currentAlbum = ref<Album | null>(null)
  const loading = ref(false)
  const total = ref(0)

  // Actions
  async function fetchAlbums(page = 1, pageSize = 20) {
    try {
      loading.value = true
      const { data } = await albumApi.getList(page, pageSize)
      albums.value = data.list
      total.value = data.total
    } finally {
      loading.value = false
    }
  }

  async function fetchAlbum(id: number) {
    try {
      loading.value = true
      const { data } = await albumApi.getDetail(id)
      currentAlbum.value = data
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

  async function deleteAlbum(id: number) {
    await albumApi.delete(id)
    albums.value = albums.value.filter(a => a.id !== id)
    total.value -= 1
    if (currentAlbum.value?.id === id) {
      currentAlbum.value = null
    }
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
    // Actions
    fetchAlbums,
    fetchAlbum,
    createAlbum,
    deleteAlbum,
    clearCurrentAlbum,
  }
})
