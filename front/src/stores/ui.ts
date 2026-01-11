import {defineStore} from 'pinia'
import {computed, ref} from 'vue'
import {createThumbnail} from "@/utils/image.ts";
import {imageApi} from "@/api/image.ts";
import {useAlbumStore} from "@/stores/album.ts";

const albumStore = useAlbumStore()

export type SortBy = 'taken_at' | 'ai_score'

export interface UploadTask {
  id: string
  file: File
  progress: number
  status: 'pending' | 'uploading' | 'success' | 'error'
  error?: string
  imageUrl?: string
  albumId?: number  // 上传成功后添加到的相册ID
  uploadedImageId?: number  // 上传成功后的图片ID
}

export const useUIStore = defineStore('ui', () => {
  // image layout State
  const gridDensity = ref(Number(localStorage.getItem("gridDensity") ?? 6)) // 1-10, 1=最密集(9列), 10=最稀疏(1列)
  // Sidebar state
  const sidebarCollapsed = ref(false)
  // Command palette state
  const commandPaletteOpen = ref(false)

  // Upload state
  const uploadDrawerOpen = ref(false)
  const uploadTasks = ref<UploadTask[]>([])

  // Selection mode state
  const isSelectionMode = ref(false)

  // Sort state
  const imageSortBy = ref<SortBy>((localStorage.getItem('imageSortBy') as SortBy) || 'taken_at')

  // Timeline state
  const timeLineState = ref<{ date: string, location: string | null } | null>(null)
  //setting tab
  const settingActiveTab = ref('security')

  // 上传完成回调
  const onUploadCompleteCallback = ref<(() => void) | null>(null)

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
  const imagePageSize = computed(() => {
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

  // Actions
  function setGridDensity(density: number) {
    gridDensity.value = Math.max(1, Math.min(10, density))
    localStorage.setItem("gridDensity", gridDensity.value + '')
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

  function addUploadTask(files: File[], albumId?: number) {
    files.forEach(file => uploadTasks.value.unshift({
      id: `${Date.now()}-${Math.random()}`,
      file,
      progress: 0,
      status: 'pending',
      albumId,
    }))
    processUploadQueue().then()
  }

  function retryUploadTask(id: string) {
    updateUploadTask(id, {
      status: 'pending',
      progress: 0,
      error: undefined,
    })
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

    // 只在有成功上传时触发回调
    if (hasSuccess && onUploadCompleteCallback.value) {
      onUploadCompleteCallback.value()
    }
  }

  async function doUploadFile(tasks: UploadTask[]): Promise<boolean[]> {
    return Promise.all(tasks.map(async task => {
      try {
        // 生成预览图 (使用缩略图以节省内存)
        updateUploadTask(task.id, {
          status: 'uploading',
        })

        // 直接在上传时传递 albumId，后端原子操作处理
        const response = await imageApi.upload(task.file, task.albumId, (progress) => {
          updateUploadTask(task.id, {progress})
        });

        const uploadedImageId = response.data.id

        // 如果指定了相册ID，更新当前相册的图片数量
        if (task.albumId && albumStore.currentAlbum?.id === task.albumId) {
          albumStore.currentAlbum.image_count += 1
        }

        // 上传成功
        updateUploadTask(task.id, {
          status: 'success',
          progress: 100,
          uploadedImageId,
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

  function setSelectionMode(mode: boolean) {
    isSelectionMode.value = mode
  }

  function setImageSortBy(sortBy: SortBy) {
    imageSortBy.value = sortBy
    localStorage.setItem('imageSortBy', sortBy)
  }

  function setTimeLineState(date: { date: string, location: string | null } | null) {
    timeLineState.value = date
  }

  function setOnUploadComplete(callback: (() => void) | null) {
    onUploadCompleteCallback.value = callback
  }

  return {
    // State
    gridDensity: gridDensity,
    sidebarCollapsed: sidebarCollapsed,
    commandPaletteOpen: commandPaletteOpen,
    uploadDrawerOpen: uploadDrawerOpen,
    uploadTasks: uploadTasks,
    isSelectionMode: isSelectionMode,
    imageSortBy: imageSortBy,
    timeLineState: timeLineState,
    settingActiveTab: settingActiveTab,

    // Computed
    gridColumns: gridColumns,
    imagePageSize: imagePageSize,
    uploadingCount: uploadingCount,
    completedCount: completedCount,
    failedCount: failedCount,
    totalProgress: totalProgress,

    // Actions
    setGridDensity: setGridDensity,
    toggleSidebar: toggleSidebar,
    openCommandPalette: openCommandPalette,
    closeCommandPalette: closeCommandPalette,
    toggleCommandPalette: toggleCommandPalette,
    openUploadDrawer: openUploadDrawer,
    closeUploadDrawer: closeUploadDrawer,
    addUploadTask: addUploadTask,
    retryUploadTask: retryUploadTask,
    removeUploadTask: removeUploadTask,
    clearCompletedTasks: clearCompletedTasks,
    setSelectionMode: setSelectionMode,
    setImageSortBy: setImageSortBy,
    setTimeLineState: setTimeLineState,
    setOnUploadComplete: setOnUploadComplete,
  }
})
