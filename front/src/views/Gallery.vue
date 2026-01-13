<template>
  <AppLayout>
    <template #header>
      <TopBar
          ref="topBarRef"
          :grid-density="uiStore.gridDensity"
          :is-selection-mode="uiStore.isSelectionMode"
          :selected-count="selectedCount"
          :show-upload="true"
          :show-sort-selector="true"
          :sort-by="uiStore.imageSortBy"
          @open-search="uiStore.openCommandPalette"
          @select-all="handleSelectAll"
          @exit-selection="exitSelectionMode"
          @density-change="uiStore.setGridDensity"
          @files-selected="handleFilesSelected"
          @sort-change="handleSortChange"
      >
        <!-- 左侧：搜索模式或默认搜索按钮 -->
        <template #left>
          <!-- 搜索模式下显示搜索状态 -->
          <div v-if="isSearchMode" class="flex items-center gap-4 animate-fade-in">
            <div
                class="flex items-center gap-3 rounded-xl bg-linear-to-r from-primary-500/20 to-pink-500/20 px-4 py-2.5 border border-primary-500/30 shadow-[0_0_15px_rgba(139,92,246,0.1)] cursor-pointer hover:bg-white/5 transition-colors group"
                @click="uiStore.openCommandPalette"
            >
              <MagnifyingGlassIcon class="h-4 w-4 text-primary-300"/>
              <span :title="searchDescription" class="text-sm font-medium text-white truncate max-w-[200px]">
                {{ searchDescription }}
              </span>
              <button
                  class="ml-2 rounded-full p-1 hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
                  title="退出搜索"
                  @click.stop="exitSearch"
              >
                <XMarkIcon class="h-4 w-4"/>
              </button>
            </div>
          </div>
          <!-- 正常模式下显示搜索按钮 -->
          <button
              v-else
              class="group flex w-75 items-center gap-3 rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-gray-400 transition-all hover:border-primary-500/30 hover:bg-white/10 hover:text-white hover:shadow-[0_0_15px_rgba(139,92,246,0.1)]"
              @click="uiStore.openCommandPalette"
          >
            <MagnifyingGlassIcon class="h-5 w-5 text-gray-500 transition-colors group-hover:text-primary-400"/>
            <span class="font-light tracking-wide">搜索影像记忆...</span>
            <div class="ml-auto flex gap-1">
              <kbd
                  class="hidden rounded bg-white/10 px-2 py-0.5 text-xs font-mono text-gray-500 group-hover:text-gray-300 md:inline-block">
                {{ isMac ? '⌘' : 'Ctrl' }}
              </kbd>
              <kbd
                  class="hidden rounded bg-white/10 px-2 py-0.5 text-xs font-mono text-gray-500 group-hover:text-gray-300 md:inline-block">
                K
              </kbd>
            </div>
          </button>
        </template>

        <!-- 选择模式下的操作按钮 -->
        <template #selection-actions>
          <button
              v-if="selectedCount > 0"
              class="flex items-center gap-2 rounded-xl bg-blue-500/10 border border-blue-500/20 px-5 py-2.5 text-sm font-medium text-blue-400 transition-all hover:bg-blue-500/20 hover:shadow-[0_0_15px_rgba(59,130,246,0.2)]"
              @click="handleBatchDownload"
          >
            <ArrowDownTrayIcon class="h-4 w-4"/>
            <span>下载 ({{ selectedCount }})</span>
          </button>
          <button
              v-if="selectedCount > 0"
              class="flex items-center gap-2 rounded-xl bg-red-500/10 border border-red-500/20 px-5 py-2.5 text-sm font-medium text-red-400 transition-all hover:bg-red-500/20 hover:shadow-[0_0_15px_rgba(239,68,68,0.2)]"
              @click="handleBatchDelete"
          >
            <TrashIcon class="h-4 w-4"/>
            <span>删除 ({{ selectedCount }})</span>
          </button>
        </template>

        <!-- 正常模式下的操作按钮 -->
        <template #actions>
          <button
              class="rounded-xl px-4 py-2.5 text-sm font-medium text-gray-400 hover:bg-white/5 hover:text-white transition-colors"
              @click="enterSelectionMode"
          >
            选择
          </button>
          <button
              class="relative flex items-center gap-2 rounded-xl bg-white px-5 py-2.5 text-sm font-bold text-black transition-all hover:shadow-[0_0_20px_rgba(255,255,255,0.3)] hover:scale-105 active:scale-95"
              @click="handleUploadClick"
          >
            <ArrowUpTrayIcon class="h-4 w-4"/>
            <span>上传</span>
          </button>
        </template>
      </TopBar>
    </template>

    <template #default>
      <!-- 命令面板 -->
      <CommandPalette/>
      <!-- 图片网格 -->
      <ImageGrid
          ref="imageGridRef"
          :fetcher="async (page, size) => (await imageApi.getList(page, size, uiStore.imageSortBy)).data"
          @update:selected-count="selectedCount = $event"
      />
    </template>

    <template #overlay>
      <!-- 悬浮时间线 -->
      <Timeline/>
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {useSearchStore} from "@/stores/search.ts";
import {computed, onUnmounted, ref, watch} from 'vue'
import {type SortBy, useUIStore} from '@/stores/ui'
import {useDialogStore} from '@/stores/dialog'
import {useScrollPosition} from '@/composables/useScrollPosition'
import {type DataSyncPayload, dataSyncService} from '@/services/dataSync'
import AppLayout from '@/components/layout/AppLayout.vue'
import CommandPalette from '@/components/search/CommandPalette.vue'
import ImageGrid from '@/components/gallery/ImageGrid.vue'
import Timeline from '@/components/gallery/Timeline.vue'
import TopBar from '@/components/layout/TopBar.vue'
import {imageApi} from '@/api/image'
import {ArrowDownTrayIcon, ArrowUpTrayIcon, MagnifyingGlassIcon, TrashIcon, XMarkIcon,} from '@heroicons/vue/24/outline'

defineOptions({name: 'GalleryView'})

const uiStore = useUIStore()
const dialogStore = useDialogStore()
const topBarRef = ref<InstanceType<typeof TopBar> | null>(null)
const imageGridRef = ref<InstanceType<typeof ImageGrid>>()

// 保存滚动位置
useScrollPosition()

function gridComponent(): InstanceType<typeof ImageGrid> {
  if (!imageGridRef.value) {
    throw new Error('ImageGrid not found')
  }
  return imageGridRef.value;
}

// 防抖定时器和待处理的图片ID集合
let debounceTimer: ReturnType<typeof setTimeout> | null = null
const pendingImageIds = new Set<number>()

async function dynamicAddImage(payload: DataSyncPayload) {
  if (!payload.ids || payload.ids.length == 0) {
    return
  }

  // 将新的图片ID添加到待处理集合
  payload.ids.forEach(id => pendingImageIds.add(id))

  // 清除之前的定时器
  if (debounceTimer) {
    clearTimeout(debounceTimer)
  }

  // 设置新的防抖定时器（300ms）
  debounceTimer = setTimeout(async () => {
    if (pendingImageIds.size === 0) return

    const idsToFetch = Array.from(pendingImageIds)
    pendingImageIds.clear()
    debounceTimer = null

    try {
      const {data: uploadedImages} = await imageApi.getByIds(idsToFetch)
      gridComponent().insertImages(uploadedImages)
    } catch (e) {
      console.error('[Gallery] 获取上传的图片失败', e)
      // 降级为全量刷新
      await gridComponent().refresh()
    }
  }, 1000)
}

// 监听数据同步事件
const unsubscribeRestored = dataSyncService.on('images:restored', dynamicAddImage)
// 监听远程上传完成事件（其他客户端上传的图片）
const unsubscribeUploaded = dataSyncService.on('images:uploaded', dynamicAddImage)

onUnmounted(() => {
  unsubscribeRestored()
  unsubscribeUploaded()
})

// 本地状态
const selectedCount = ref(0)
const isSearchMode = ref(false)
const searchDescription = ref('')

const isMac = computed(() => navigator.userAgent.includes('Mac'))

useSearchStore().subsribe("gallary", (desc, fetcher) => {
  searchDescription.value = desc
  isSearchMode.value = true
  gridComponent().refresh(50, fetcher)
})

function exitSearch() {
  isSearchMode.value = false
  searchDescription.value = ''
  gridComponent().refresh()
}

// 排序变更
function handleSortChange(sortBy: SortBy) {
  uiStore.setImageSortBy(sortBy)
  // 如果不在搜索模式，刷新列表
  if (!isSearchMode.value) {
    gridComponent().refresh()
  }
}

// 监听排序变化，自动刷新（当从其他地方修改时）
watch(() => uiStore.imageSortBy, () => {
  if (!isSearchMode.value) {
    gridComponent().refresh()
  }
})

// 选择模式
function enterSelectionMode() {
  uiStore.setSelectionMode(true)
}

function exitSelectionMode() {
  uiStore.setSelectionMode(false)
  gridComponent().clearSelection()
}

function handleSelectAll() {
  const grid = gridComponent()
  if (!grid) return

  const images = grid.images
  const isAllSelected = images.length > 0 && selectedCount.value === images.filter(i => i !== null).length
  if (isAllSelected) {
    grid.clearSelection()
  } else {
    grid.selectAll()
  }
}

// 批量操作
async function handleBatchDelete() {
  const confirmed = await dialogStore.confirm({
    title: '确认删除',
    message: `确定要删除选中的 ${selectedCount.value} 张图片吗？`,
    type: 'warning',
    confirmText: '删除'
  })

  if (!confirmed) return

  try {
    await gridComponent().deleteBatch()
    if (gridComponent().images.length === 0) {
      exitSelectionMode()
    }
  } catch (error) {
    console.error('批量删除失败', error)
    dialogStore.alert({
      title: '删除失败',
      message: '批量删除过程中发生错误，请重试。',
      type: 'error'
    })
  }
}

function handleBatchDownload() {
  if (selectedCount.value === 0) return
  const ids = gridComponent().selectedIds
  if (ids) imageApi.downloadZipped(ids)
}

// 上传
function handleUploadClick() {
  topBarRef.value?.triggerUpload()
}

function handleFilesSelected(files: File[]) {
  uiStore.addUploadTask(files)
  uiStore.openUploadDrawer()
}
</script>
