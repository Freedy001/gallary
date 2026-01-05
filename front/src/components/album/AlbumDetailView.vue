<template>
  <AppLayout>
    <template #header>
      <TopBar
        ref="topBarRef"
        :grid-density="uiStore.gridDensity"
        :is-selection-mode="uiStore.isSelectionMode"
        :selected-count="imageStore.selectedCount"
        :show-upload="true"
        :total-count="imageStore.images.length"
        :upload-album-id="currentAlbumId"
        @select-all="handleSelectAll"
        @exit-selection="exitSelectionMode"
        @density-change="uiStore.setGridDensity"
        @files-selected="handleFilesSelected"
      >
        <!-- 自定义左侧：面包屑导航 -->
        <template #left>
          <Breadcrumb :items="breadcrumbItems"/>
        </template>

        <!-- 自定义选择模式操作：从相册移除 -->
        <template #selection-actions>
          <button
              :disabled="imageStore.selectedCount === 0"
              @click="handleRemoveFromAlbum"
              class="flex items-center gap-2 rounded-xl bg-red-500/10 border border-red-500/20 px-5 py-2.5 text-sm font-medium text-red-400 transition-all hover:bg-red-500/20 hover:shadow-[0_0_15px_rgba(239,68,68,0.2)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-none"
          >
            <MinusCircleIcon class="h-4 w-4"/>
            <span>从相册移除 ({{ imageStore.selectedCount }})</span>
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
      <ImageGrid mode="gallery" :exclude-album-id="currentAlbumId"/>
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {computed, onMounted, onUnmounted, ref} from 'vue'
import {useImageStore} from '@/stores/image'
import {useUIStore} from '@/stores/ui'
import {useAlbumStore} from '@/stores/album'
import {albumApi} from '@/api/album'
import {emptyPage, type Image, type Pageable} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TopBar from '@/components/layout/TopBar.vue'
import ImageGrid from '@/components/gallery/ImageGrid.vue'
import Breadcrumb from '@/components/common/Breadcrumb.vue'
import CommandPalette from '@/components/search/CommandPalette.vue'
import {ArrowUpTrayIcon, MinusCircleIcon} from '@heroicons/vue/24/outline'

const imageStore = useImageStore()
const uiStore = useUIStore()
const albumStore = useAlbumStore()
const topBarRef = ref<InstanceType<typeof TopBar> | null>(null)
const currentAlbumId = computed(() => albumStore.currentAlbum?.id)

const breadcrumbItems = computed(() => [
  { label: '相册', to: '/gallery/albums' },
  { label: albumStore.currentAlbum?.name || '加载中...' }
])

// 选择模式
function enterSelectionMode() {
  uiStore.setSelectionMode(true)
}

function exitSelectionMode() {
  uiStore.setSelectionMode(false)
  imageStore.clearSelection()
}

function handleSelectAll() {
  const isAllSelected = imageStore.images.length > 0 && imageStore.selectedCount === imageStore.images.length
  if (isAllSelected) {
    imageStore.clearSelection()
  } else {
    imageStore.images.forEach(img => img && imageStore.selectImage(img.id))
  }
}

// 从相册移除
async function handleRemoveFromAlbum() {
  const ids = Array.from(imageStore.selectedImages)
  if (ids.length === 0) return

  if (!currentAlbumId.value) return

  try {
    await albumApi.removeImages(currentAlbumId.value, ids)
    // 从列表移除
    imageStore.images = imageStore.images.filter(img => img === null || !ids.includes(img.id))
    imageStore.total -= ids.length
    imageStore.clearSelection()

    // 更新相册信息
    if (albumStore.currentAlbum) {
      albumStore.currentAlbum.image_count -= ids.length
    }

    // 如果没有图片了，退出选择模式
    if (imageStore.total === 0) {
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

onMounted(async () => {
  // 使用相册图片 API 作为数据源
  const pageSize = uiStore.imagePageSize
  await imageStore.refreshImages(async (page: number, size: number): Promise<Pageable<Image>> => {
    return currentAlbumId.value ? (await albumApi.getImages(currentAlbumId.value, page, size)).data : emptyPage
  }, pageSize)
})

onUnmounted(() => {
  imageStore.clearSelection()
  uiStore.setSelectionMode(false)
  albumStore.clearCurrentAlbum()
})
</script>
