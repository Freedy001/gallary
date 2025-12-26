<template>
  <AppLayout>
    <template #header>
      <TopBar :upload-album-id="currentAlbumId">
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
import {computed, onMounted, onUnmounted} from 'vue'
import {useImageStore} from '@/stores/image.ts'
import {useUIStore} from '@/stores/ui.ts'
import {useAlbumStore} from '@/stores/album.ts'
import {albumApi} from '@/api/album.ts'
import {emptyPage, type Image, type Pageable} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TopBar from '@/components/layout/TopBar.vue'
import ImageGrid from '@/components/gallery/ImageGrid.vue'
import Breadcrumb from '@/components/common/Breadcrumb.vue'
import CommandPalette from '@/components/search/CommandPalette.vue'
import {MinusCircleIcon} from '@heroicons/vue/24/outline'

const imageStore = useImageStore()
const uiStore = useUIStore()
const albumStore = useAlbumStore()
const currentAlbumId = computed(() => albumStore.currentAlbum?.id)

const breadcrumbItems = computed(() => [
  {label: '相册', to: '/gallery/albums'},
  {label: albumStore.currentAlbum?.name || '加载中...'}
])

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
      uiStore.setSelectionMode(false)
    }
  } catch (err) {
    console.error('从相册移除失败', err)
  }
}

onMounted(async () => {
  // 使用相册图片 API 作为数据源
  const pageSize = uiStore.pageSize
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
