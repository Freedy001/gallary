import {defineStore} from 'pinia'
import {ref, computed} from 'vue'

export interface UploadTask {
  id: string
  file: File
  progress: number
  status: 'pending' | 'uploading' | 'success' | 'error'
  error?: string
  imageUrl?: string
}

export const useUIStore = defineStore('ui', () => {
  // image layout State
  const gridDensity = ref(6) // 1-10, 1=最密集(9列), 10=最稀疏(1列)
  // Sidebar state
  const sidebarCollapsed = ref(false)
  // Command palette state
  const commandPaletteOpen = ref(false)
  // Image viewer state
  const imageViewerOpen = ref(false)

  // Upload state
  const uploadDrawerOpen = ref(false)
  const uploadTasks = ref<UploadTask[]>([])

  // Loading state
  const globalLoading = ref(false)
  const loadingMessage = ref('')

  // Selection mode state
  const isSelectionMode = ref(false)

  // Timeline state
  const timeLineState = ref<{ date: string, location: string | null } | null>(null)

  // Computed
  const gridColumns = computed(() => {
    const desktopColumns = {
      1: 9,  // Grid
      2: 8,  // Grid
      3: 7,  // Grid
      4: 6,  // Grid
      5: 5,  // Grid
      6: 4,  // Grid (Default)
      7: 3,  // Grid
      8: 3,  // Waterfall
      9: 2,  // Waterfall
      10: 1, // Waterfall
    }[gridDensity.value] || 4

    return {
      desktop: desktopColumns,
      tablet: Math.max(2, Math.ceil(desktopColumns / 2)),
      mobile: desktopColumns >= 4 ? 2 : 1,
    }
  })

  const uploadingCount = computed(() =>
    uploadTasks.value.filter(t => t.status === 'uploading').length
  )

  const completedCount = computed(() =>
    uploadTasks.value.filter(t => t.status === 'success').length
  )

  const failedCount = computed(() =>
    uploadTasks.value.filter(t => t.status === 'error').length
  )

  const totalProgress = computed(() => {
    if (uploadTasks.value.length === 0) return 0
    const total = uploadTasks.value.reduce((sum, task) => sum + task.progress, 0)
    return Math.round(total / uploadTasks.value.length)
  })

  const hasActiveUploads = computed(() =>
    uploadTasks.value.some(t => t.status === 'uploading' || t.status === 'pending')
  )

  // Actions
  function setGridDensity(density: number) {
    gridDensity.value = Math.max(1, Math.min(10, density))
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function openCommandPalette() {
    commandPaletteOpen.value = true
  }

  function closeCommandPalette() {
    commandPaletteOpen.value = false
  }

  function toggleCommandPalette() {
    commandPaletteOpen.value = !commandPaletteOpen.value
  }

  function openUploadDrawer() {
    uploadDrawerOpen.value = true
  }

  function closeUploadDrawer() {
    uploadDrawerOpen.value = false
  }

  function toggleUploadDrawer() {
    uploadDrawerOpen.value = !uploadDrawerOpen.value
  }

  function addUploadTask(file: File): UploadTask {
    const task: UploadTask = {
      id: `${Date.now()}-${Math.random()}`,
      file,
      progress: 0,
      status: 'pending',
    }
    uploadTasks.value.push(task)
    return task
  }

  function updateUploadTask(id: string, updates: Partial<UploadTask>) {
    const task = uploadTasks.value.find(t => t.id === id)
    if (task) {
      Object.assign(task, updates)
    }
  }

  function removeUploadTask(id: string) {
    const index = uploadTasks.value.findIndex(t => t.id === id)
    if (index !== -1) {
      uploadTasks.value.splice(index, 1)
    }
  }

  function clearCompletedTasks() {
    uploadTasks.value = uploadTasks.value.filter(
      t => t.status !== 'success' && t.status !== 'error'
    )
  }

  function clearAllTasks() {
    uploadTasks.value = []
  }

  function setGlobalLoading(loading: boolean, message = '') {
    globalLoading.value = loading
    loadingMessage.value = message
  }

  function setSelectionMode(mode: boolean) {
    isSelectionMode.value = mode
  }

  function setTimeLineState(date: { date: string, location: string | null } | null) {
    timeLineState.value = date
  }

  return {
    // State
    gridDensity,
    sidebarCollapsed,
    commandPaletteOpen,
    imageViewerOpen,
    uploadDrawerOpen,
    uploadTasks,
    globalLoading,
    loadingMessage,
    isSelectionMode,
    timeLineState,

    // Computed
    gridColumns,
    uploadingCount,
    completedCount,
    failedCount,
    totalProgress,
    hasActiveUploads,

    // Actions
    setGridDensity,
    toggleSidebar,
    openCommandPalette,
    closeCommandPalette,
    toggleCommandPalette,
    openUploadDrawer,
    closeUploadDrawer,
    toggleUploadDrawer,
    addUploadTask,
    updateUploadTask,
    removeUploadTask,
    clearCompletedTasks,
    clearAllTasks,
    setGlobalLoading,
    setSelectionMode,
    setTimeLineState,
  }
})
