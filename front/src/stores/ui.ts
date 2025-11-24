import {defineStore} from 'pinia'
import {ref, computed} from 'vue'
import {createThumbnail} from "@/utils/image.ts";
import {imageApi} from "@/api/image.ts";
import {useImageStore} from "@/stores/image.ts";

const imageStore = useImageStore()

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

  // 根据网格列数动态计算每页应加载的图片数量
  // 确保至少加载足够填满 2 屏的图片，以便触发滚动
  const pageSize = computed(() => {
    const cols = gridColumns.value.desktop
    // 估算每屏能显示的行数（假设视口高度约 900px，每个图片约 200px）
    const rowsPerScreen = Math.ceil(900 / 200)
    // 每页至少加载 2 屏的图片数量，最少 20 张
    const minSize = Math.max(20, cols * rowsPerScreen * 2)
    // 向上取整到 10 的倍数，便于分页计算
    return Math.ceil(minSize / 10) * 10
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

  function addUploadTask(files: File[]) {
    files.forEach(file => uploadTasks.value.unshift({
      id: `${Date.now()}-${Math.random()}`,
      file,
      progress: 0,
      status: 'pending',
    }))
    processUploadQueue().then()
  }

  // 处理上传队列
  async function processUploadQueue() {
    const tasks = uploadTasks.value
    const pendingTasks = tasks.filter(t => t.status === 'pending')

    // 没有待处理任务时直接返回
    if (pendingTasks.length === 0) return

    // 异步生成缩略图，只为没有缩略图的任务生成
    tasks.forEach(task => {
      if (!task.imageUrl) {
        createThumbnail(task.file).then(imageUrl => {
          if (imageUrl) updateUploadTask(task.id, {imageUrl})
        }).catch(console.error)
      }
    })

    // 计算批次数量（向上取整）
    const turn = Math.ceil(pendingTasks.length / 5)
    let hasSuccess = false

    for (let i = 0; i < turn; i++) {
      const tasksToStart = pendingTasks.slice(
          i * 5,
          (i + 1) * 5
      )
      const results = await doUploadFile(tasksToStart)
      if (results.some(r => r)) hasSuccess = true
    }

    // 只在有成功上传时刷新图片列表
    if (hasSuccess) {
      await imageStore.refreshImages()
    }
  }

  async function doUploadFile(tasks: UploadTask[]): Promise<boolean[]> {
    return Promise.all(tasks.map(async task => {
      try {
        // 生成预览图 (使用缩略图以节省内存)
        updateUploadTask(task.id, {
          status: 'uploading',
        })

        await imageApi.upload(task.file, (progress) => {
          updateUploadTask(task.id, {progress})
        });

        // 上传成功
        updateUploadTask(task.id, {
          status: 'success',
          progress: 100,
        })
        return true
      } catch (error) {
        // 上传失败
        updateUploadTask(task.id, {
          status: 'error',
          error: error instanceof Error ? error.message : '上传失败',
        })
        return false
      }
    }))
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
    uploadDrawerOpen,
    uploadTasks,
    globalLoading,
    loadingMessage,
    isSelectionMode,
    timeLineState,

    // Computed
    gridColumns,
    pageSize,
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
    addUploadTask,
    updateUploadTask,
    removeUploadTask,
    clearCompletedTasks,
    setGlobalLoading,
    setSelectionMode,
    setTimeLineState,
  }
})
