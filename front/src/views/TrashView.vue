<template>
  <AppLayout>
    <template #header>
      <div class="relative z-30 flex h-20 w-full items-center justify-between border-b border-white/5 bg-transparent px-8 transition-all duration-300 backdrop-blur-sm">
        <!-- 左侧区域 -->
        <div class="flex items-center gap-4">
          <!-- 正常模式：标题 -->
          <div v-if="!uiStore.isSelectionMode" class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-white/5 text-primary-400 ring-1 ring-white/10">
              <TrashIcon class="h-5 w-5"/>
            </div>
            <div class="flex flex-col">
              <h1 class="text-lg font-medium text-white leading-tight">最近删除</h1>
              <span class="text-xs text-gray-500 font-mono mt-0.5">{{ imageStore.total }} 项资源</span>
            </div>
          </div>

          <!-- 选择模式：计数器 + 全选 (参考 TopBar) -->
          <div v-else class="flex items-center gap-6 animate-fade-in">
            <div class="flex items-center gap-3 rounded-lg bg-primary-500/10 px-4 py-2 border border-primary-500/20">
              <span class="h-2 w-2 rounded-full bg-primary-500 animate-pulse"></span>
              <span class="text-lg font-medium text-white">已选择 {{ imageStore.selectedCount }} 项</span>
            </div>
            <button
                @click="toggleSelectAll"
                class="text-sm font-medium text-primary-400 hover:text-primary-300 hover:underline underline-offset-4"
            >
              {{ isAllSelected ? '取消全选' : '全选所有' }}
            </button>
          </div>
        </div>

        <!-- 右侧操作区域 -->
        <div class="flex items-center gap-6">
          <!-- 选择模式下的操作按钮 -->
          <div v-if="uiStore.isSelectionMode" class="flex items-center gap-4">
            <button
                :disabled="imageStore.selectedCount === 0"
                @click="handleBatchRestore"
                class="flex items-center gap-2 rounded-xl bg-blue-500/10 border border-blue-500/20 px-5 py-2.5 text-sm font-medium text-blue-400 transition-all hover:bg-blue-500/20 hover:shadow-[0_0_15px_rgba(59,130,246,0.2)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-none"
            >
              <ArrowUturnLeftIcon class="h-4 w-4" />
              <span>恢复 ({{ imageStore.selectedCount }})</span>
            </button>

            <button
                :disabled="imageStore.selectedCount === 0"
                @click="handleBatchPermanentDelete"
                class="flex items-center gap-2 rounded-xl bg-red-500/10 border border-red-500/20 px-5 py-2.5 text-sm font-medium text-red-400 transition-all hover:bg-red-500/20 hover:shadow-[0_0_15px_rgba(239,68,68,0.2)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-none"
            >
              <XMarkIcon class="h-4 w-4" />
              <span>彻底删除 ({{ imageStore.selectedCount }})</span>
            </button>

            <button
                @click="exitSelectionMode"
                class="rounded-xl border border-white/10 bg-white/5 px-6 py-2.5 text-sm font-medium text-white hover:bg-white/10 transition-colors"
            >
              完成
            </button>
          </div>

          <!-- 正常模式下的操作按钮 -->
          <div v-else-if="validImagesCount > 0" class="flex items-center gap-4">
            <button
                @click="enterSelectionMode"
                class="rounded-xl px-4 py-2.5 text-sm font-medium text-gray-400 hover:bg-white/5 hover:text-white transition-colors"
            >
              选择
            </button>
          </div>
        </div>
      </div>
    </template>

    <template #default>
      <ImageGrid mode="trash" />
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useImageStore } from '@/stores/image'
import { useUIStore } from '@/stores/ui'
import { useDialogStore } from '@/stores/dialog'
import { useStorageStore } from '@/stores/storage'
import { imageApi } from '@/api/image'
import type { Image, Pageable } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import ImageGrid from '@/components/gallery/ImageGrid.vue'
import {
  TrashIcon,
  ArrowUturnLeftIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'

const imageStore = useImageStore()
const uiStore = useUIStore()
const dialogStore = useDialogStore()
const storageStore = useStorageStore()

// 计算有效的（已加载的）图片数量
const validImagesCount = computed(() => {
  return imageStore.images.filter(img => img !== null).length
})

const isAllSelected = computed(() => {
  const validImages = imageStore.images.filter(img => img !== null) as Image[]
  return validImages.length > 0 && validImages.every(img => imageStore.selectedImages.has(img.id))
})

function toggleSelectAll() {
  if (isAllSelected.value) {
    imageStore.clearSelection()
  } else {
    imageStore.images.forEach(img => {
      if (img) imageStore.selectImage(img.id)
    })
  }
}

function enterSelectionMode() {
  uiStore.setSelectionMode(true)
}

function exitSelectionMode() {
  uiStore.setSelectionMode(false)
  imageStore.clearSelection()
}

async function handleBatchRestore() {
  const ids = Array.from(imageStore.selectedImages)
  if (ids.length === 0) return

  try {
    await imageApi.restoreImages(ids)
    // 从列表移除已恢复的图片
    imageStore.images = imageStore.images.filter(img => img === null || !imageStore.selectedImages.has(img.id))
    imageStore.total -= ids.length
    imageStore.clearSelection()

    // 恢复后更新侧边栏显示的总数（增加）
    storageStore.updateTotalImages(ids.length)

    // 如果没有图片了，退出选择模式
    if (imageStore.total === 0) {
      uiStore.setSelectionMode(false)
    }
  } catch (err) {
    console.error('批量恢复图片失败', err)
  }
}

async function handleBatchPermanentDelete() {
  const ids = Array.from(imageStore.selectedImages)
  if (ids.length === 0) return

  const confirmed = await dialogStore.confirm({
    title: '确认彻底删除',
    message: `确定要彻底删除 ${ids.length} 张图片吗？此操作不可恢复。`,
    type: 'error',
    confirmText: '彻底删除'
  })

  if (!confirmed) return

  try {
    await imageApi.permanentlyDelete(ids)
    // 从列表移除已删除的图片
    imageStore.images = imageStore.images.filter(img => img === null || !imageStore.selectedImages.has(img.id))
    imageStore.total -= ids.length
    imageStore.clearSelection()

    // 如果没有图片了，退出选择模式
    if (imageStore.total === 0) {
      uiStore.setSelectionMode(false)
    }
  } catch (err) {
    console.error('批量彻底删除图片失败', err)
  }
}

onMounted(async () => {
  // 使用 refreshImages 切换数据源为回收站 API
  const pageSize = uiStore.pageSize
  await imageStore.refreshImages(async (page: number, size: number): Promise<Pageable<Image>> => {
    return (await imageApi.getDeletedList(page, size)).data
  }, pageSize)

  // 加载第一页
  if (imageStore.images.length === 0 && imageStore.total > 0) {
     // Note: refreshImages already calls fetchImages(1) internally
  } else if (imageStore.images.length === 0) {
      // refreshImages calls fetchImages(1) which populates total.
      // If total was 0 initially, fine.
  }
})

onUnmounted(() => {
  imageStore.clearSelection()
  uiStore.setSelectionMode(false)
  // 注意：不需要重置 fetcher，因为 Gallery.vue 会在挂载时重置它
})
</script>
