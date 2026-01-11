<template>
  <AppLayout>
    <template #header>
      <TopBar
          ref="topBarRef"
          :grid-density="uiStore.gridDensity"
          :is-selection-mode="uiStore.isSelectionMode"
          :selected-count="selectedCount"
          :show-upload="true"
          :upload-album-id="currentAlbumId"
          :show-sort-selector="true"
          :sort-by="uiStore.imageSortBy"
          @select-all="handleSelectAll"
          @exit-selection="exitSelectionMode"
          @density-change="uiStore.setGridDensity"
          @files-selected="handleFilesSelected"
          @sort-change="handleSortChange"
      >
        <!-- 自定义左侧：面包屑导航 -->
        <template #left>
          <Breadcrumb :items="breadcrumbItems"/>
        </template>

        <!-- 自定义选择模式操作：从相册移除 -->
        <template #selection-actions>
          <button
              :disabled="selectedCount === 0"
              @click="handleRemoveFromAlbum"
              class="flex items-center gap-2 rounded-xl bg-red-500/10 border border-red-500/20 px-5 py-2.5 text-sm font-medium text-red-400 transition-all hover:bg-red-500/20 hover:shadow-[0_0_15px_rgba(239,68,68,0.2)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-none"
          >
            <MinusCircleIcon class="h-4 w-4"/>
            <span>从相册移除 ({{ selectedCount }})</span>
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
      <ImageGrid
          ref="imageGridRef"
          :album-id="currentAlbumId"
          :fetcher=" async (page, size) => currentAlbumId ? (await albumApi.getImages(currentAlbumId, page, size, uiStore.imageSortBy)).data : emptyPage"
          mode="gallery"
          @update:selected-count="selectedCount = $event"
      />
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {computed, onUnmounted, ref, watch} from 'vue'
import {type SortBy, useUIStore} from '@/stores/ui'
import {useAlbumStore} from '@/stores/album'
import {albumApi} from '@/api/album'
import {emptyPage} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TopBar from '@/components/layout/TopBar.vue'
import ImageGrid from '@/components/gallery/ImageGrid.vue'
import Breadcrumb from '@/components/common/Breadcrumb.vue'
import {ArrowUpTrayIcon, MinusCircleIcon} from '@heroicons/vue/24/outline'

const uiStore = useUIStore()
const albumStore = useAlbumStore()
const topBarRef = ref<InstanceType<typeof TopBar> | null>(null)
const imageGridRef = ref<InstanceType<typeof ImageGrid> | null>(null)
const currentAlbumId = computed(() => albumStore.currentAlbum?.id)

function gridComponent(): InstanceType<typeof ImageGrid> {
  if (!imageGridRef.value) throw new Error('ImageGrid is not ready')
  return imageGridRef.value;
}

// 本地状态
const selectedCount = ref(0)

const breadcrumbItems = computed(() => [
  {label: '相册', to: '/gallery/albums'},
  {label: albumStore.currentAlbum?.name || '加载中...'}
])

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

// 排序变更
function handleSortChange(sortBy: SortBy) {
  uiStore.setImageSortBy(sortBy)
  gridComponent().refresh()
}

// 监听排序变化
watch(() => uiStore.imageSortBy, () => {
  gridComponent().refresh()
})

// 从相册移除
async function handleRemoveFromAlbum() {
  const ids = gridComponent().selectedIds
  if (!ids || ids.length === 0) return
  if (!currentAlbumId.value) return

  try {
    await albumApi.removeImages(currentAlbumId.value, ids)
    // 刷新列表
    await gridComponent().refresh()
    gridComponent().clearSelection()

    // 更新相册信息
    if (albumStore.currentAlbum) {
      albumStore.currentAlbum.image_count -= ids.length
    }

    // 如果没有图片了，退出选择模式
    if (gridComponent().images.length === 0) {
      exitSelectionMode()
    }
  } catch (err) {
    console.error('从相册移除失败', err)
  }
}

// 上传
function handleUploadClick() {
  topBarRef.value?.triggerUpload()
}

function handleFilesSelected(files: File[], albumId?: number) {
  uiStore.addUploadTask(files, albumId)
  uiStore.openUploadDrawer()
}

onUnmounted(() => {
  uiStore.setSelectionMode(false)
  albumStore.clearCurrentAlbum()
})
</script>
