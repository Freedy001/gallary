import {defineStore} from 'pinia'
import {computed, ref} from 'vue'
import {albumApi} from '@/api/album'
import type {Album} from '@/types'

// 分区状态类型
interface AlbumSection {
  albums: (Album | null)[]
  total: number
  loading: boolean
  loadingPages: Set<number>
}

function createEmptySection(): AlbumSection {
  return {
    albums: [],
    total: 0,
    loading: false,
    loadingPages: new Set()
  }
}

export const useAlbumStore = defineStore('album', () => {
  // 分区状态
  const normalSection = ref<AlbumSection>(createEmptySection())
  const smartSection = ref<AlbumSection>(createEmptySection())

  // 其他状态
  const currentAlbum = ref<Album | null>(null)
  const selectedAlbums = ref<Set<number>>(new Set())

  // 向后兼容的计算属性
  const albums = computed(() => [
    ...normalSection.value.albums.filter((a): a is Album => a !== null),
    ...smartSection.value.albums.filter((a): a is Album => a !== null)
  ])
  const total = computed(() => normalSection.value.total + smartSection.value.total)
  const loading = computed(() => normalSection.value.loading || smartSection.value.loading)

  // Computed
  const selectedCount = computed(() => selectedAlbums.value.size)
  const hasSelection = computed(() => selectedAlbums.value.size > 0)
  const selectedAlbumList = computed(() =>
    albums.value.filter(a => selectedAlbums.value.has(a.id))
  )

  // Actions
  async function fetchSection(section: 'normal' | 'smart', page = 1, pageSize = 20) {
    const sectionRef = section === 'normal' ? normalSection : smartSection
    const isSmart = section === 'smart'

    // 防止重复加载同一页
    if (sectionRef.value.loadingPages.has(page)) return

    try {
      sectionRef.value.loadingPages.add(page)
      if (page === 1) sectionRef.value.loading = true

      const {data} = await albumApi.getList({page, pageSize, isSmart})

      if (page === 1) {
        // 初始化占位符数组
        sectionRef.value.albums = new Array(data.total).fill(null)
        sectionRef.value.total = data.total
      } else {
        // 确保数组长度足够
        if (sectionRef.value.albums.length < data.total) {
          const diff = data.total - sectionRef.value.albums.length
          for (let i = 0; i < diff; i++) sectionRef.value.albums.push(null)
        }
      }

      // 填充数据
      const startIndex = (page - 1) * pageSize
      data.list.forEach((album, i) => {
        if (startIndex + i < sectionRef.value.albums.length) {
          sectionRef.value.albums[startIndex + i] = album
        }
      })
    } finally {
      sectionRef.value.loadingPages.delete(page)
      if (page === 1) sectionRef.value.loading = false
    }
  }

  // 刷新所有分区（并行加载第一页）
  async function refreshAlbums(pageSize = 20) {
    await Promise.all([
      fetchSection('normal', 1, pageSize),
      fetchSection('smart', 1, pageSize)
    ])
  }

  // 向后兼容的 fetchAlbums（内部调用 refreshAlbums）
  async function fetchAlbums(page = 1, pageSize = 20) {
    if (page === 1) {
      await refreshAlbums(pageSize)
    }
  }

  async function createAlbum(name: string, description?: string) {
    const {data} = await albumApi.create({name, description})
    // 普通相册添加到 normalSection 开头
    normalSection.value.albums.unshift(data)
    normalSection.value.total += 1
    return data
  }

  async function updateAlbum(id: number, name: string, description?: string) {
    const {data} = await albumApi.update(id, {name, description})
    // 在两个分区中查找并更新
    const normalIndex = normalSection.value.albums.findIndex(a => a?.id === id)
    if (normalIndex !== -1) {
      normalSection.value.albums[normalIndex] = {...normalSection.value.albums[normalIndex]!, ...data}
    }
    const smartIndex = smartSection.value.albums.findIndex(a => a?.id === id)
    if (smartIndex !== -1) {
      smartSection.value.albums[smartIndex] = {...smartSection.value.albums[smartIndex]!, ...data}
    }
    if (currentAlbum.value?.id === id) {
      currentAlbum.value = {...currentAlbum.value, ...data}
    }
    return data
  }

  async function setAlbumCover(id: number, imageId: number) {
    await albumApi.setCover(id, imageId)
  }

  async function deleteAlbum(ids: number | number[]) {
    const idArray = Array.isArray(ids) ? ids : [ids]
    await albumApi.delete(idArray)
    // 从两个分区中移除
    normalSection.value.albums = normalSection.value.albums.filter(a => a === null || !idArray.includes(a.id))
    smartSection.value.albums = smartSection.value.albums.filter(a => a === null || !idArray.includes(a.id))
    // 更新 total
    normalSection.value.total = normalSection.value.albums.filter(a => a !== null).length
    smartSection.value.total = smartSection.value.albums.filter(a => a !== null).length
    // 清除选中状态
    idArray.forEach(id => selectedAlbums.value.delete(id))
    if (currentAlbum.value && idArray.includes(currentAlbum.value.id)) {
      currentAlbum.value = null
    }
  }

  async function deleteSelectedAlbums() {
    const ids = Array.from(selectedAlbums.value)
    await deleteAlbum(ids)
  }

  // 复制相册
  async function copyAlbum(ids: number | number[]) {
    const idArray = Array.isArray(ids) ? ids : [ids]
    await albumApi.copy(idArray)
    // 刷新相册列表
    await refreshAlbums()
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
    // 分区状态
    normalSection: normalSection,
    smartSection: smartSection,
    // 向后兼容的状态
    albums: albums,
    currentAlbum: currentAlbum,
    loading: loading,
    total: total,
    selectedAlbums: selectedAlbums,
    // Computed
    selectedCount: selectedCount,
    hasSelection: hasSelection,
    selectedAlbumList: selectedAlbumList,
    // Actions
    fetchSection: fetchSection,
    refreshAlbums: refreshAlbums,
    fetchAlbums: fetchAlbums,
    createAlbum: createAlbum,
    updateAlbum: updateAlbum,
    setAlbumCover: setAlbumCover,
    deleteAlbum: deleteAlbum,
    deleteSelectedAlbums: deleteSelectedAlbums,
    copyAlbum: copyAlbum,
    toggleAlbumSelection: toggleAlbumSelection,
    clearSelection: clearSelection,
    selectAll: selectAll,
    clearCurrentAlbum: clearCurrentAlbum,
  }
})
