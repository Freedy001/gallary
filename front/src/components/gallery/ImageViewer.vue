<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="uiStore.imageViewerOpen && currentImage"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black"
        @click.self="close"
      >
        <!-- 关闭按钮 -->
        <button
          @click="close"
          class="absolute right-4 top-4 z-10 rounded-full bg-black bg-opacity-50 p-2 text-white transition-colors hover:bg-opacity-70"
        >
          <XMarkIcon class="h-6 w-6" />
        </button>

        <!-- 上一张 -->
        <button
          v-if="hasPrevious"
          @click="previous"
          class="absolute left-4 top-1/2 z-10 -translate-y-1/2 rounded-full bg-black bg-opacity-50 p-3 text-white transition-colors hover:bg-opacity-70"
        >
          <ChevronLeftIcon class="h-6 w-6" />
        </button>

        <!-- 下一张 -->
        <button
          v-if="hasNext"
          @click="next"
          class="absolute right-4 top-1/2 z-10 -translate-y-1/2 rounded-full bg-black bg-opacity-50 p-3 text-white transition-colors hover:bg-opacity-70"
        >
          <ChevronRightIcon class="h-6 w-6" />
        </button>

        <!-- 图片容器 -->
        <div
          ref="imageContainerRef"
          class="relative h-full w-full overflow-hidden"
          @wheel.prevent="handleWheel"
        >
          <img
            :src="imageUrl"
            :alt="currentImage.original_name"
            class="absolute left-1/2 top-1/2 max-h-full max-w-full -translate-x-1/2 -translate-y-1/2 object-contain transition-transform duration-200"
            :style="{
              transform: `translate(-50%, -50%) scale(${scale})`,
            }"
            @load="handleImageLoad"
          />
        </div>

        <!-- 底部工具栏 -->
        <div class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/80 to-transparent p-6">
          <div class="mx-auto max-w-4xl">
            <!-- 文件信息 -->
            <div class="mb-4 text-white">
              <h3 class="text-lg font-semibold">{{ currentImage.original_name }}</h3>
              <div class="mt-2 flex flex-wrap gap-4 text-sm text-gray-300">
                <span v-if="currentImage.taken_at">{{ formatDate(currentImage.taken_at) }}</span>
                <span v-if="currentImage.camera_model">{{ currentImage.camera_model }}</span>
                <span>{{ currentImage.width }} × {{ currentImage.height }}</span>
                <span>{{ formatFileSize(currentImage.file_size) }}</span>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex items-center gap-2">
              <button
                @click="zoomOut"
                class="rounded-lg bg-white/20 px-3 py-2 text-sm text-white hover:bg-white/30"
              >
                <MinusIcon class="h-4 w-4" />
              </button>
              <span class="px-3 text-sm text-white">{{ Math.round(scale * 100) }}%</span>
              <button
                @click="zoomIn"
                class="rounded-lg bg-white/20 px-3 py-2 text-sm text-white hover:bg-white/30"
              >
                <PlusIcon class="h-4 w-4" />
              </button>
              <button
                @click="resetZoom"
                class="rounded-lg bg-white/20 px-3 py-2 text-sm text-white hover:bg-white/30"
              >
                重置
              </button>

              <div class="flex-1" />

              <button
                @click="downloadImage"
                class="flex items-center gap-2 rounded-lg bg-white/20 px-4 py-2 text-sm text-white hover:bg-white/30"
              >
                <ArrowDownTrayIcon class="h-4 w-4" />
                下载
              </button>

              <button
                @click="deleteImage"
                class="flex items-center gap-2 rounded-lg bg-red-600/80 px-4 py-2 text-sm text-white hover:bg-red-600"
              >
                <TrashIcon class="h-4 w-4" />
                删除
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { useUIStore } from '@/stores/ui'
import { useImageStore } from '@/stores/image'
import { imageApi } from '@/api/image'
import {
  XMarkIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  MinusIcon,
  PlusIcon,
  ArrowDownTrayIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'

const uiStore = useUIStore()
const imageStore = useImageStore()

const imageContainerRef = ref<HTMLElement>()
const scale = ref(1)

const currentImage = computed(() => {
  const index = uiStore.currentViewerIndex
  return imageStore.images[index] || null
})

const imageUrl = computed(() => {
  if (!currentImage.value) return ''
  return imageApi.getImageUrl(currentImage.value.storage_path)
})

const hasPrevious = computed(() => uiStore.currentViewerIndex > 0)
const hasNext = computed(() => uiStore.currentViewerIndex < imageStore.images.length - 1)

function close() {
  uiStore.closeImageViewer()
  resetZoom()
}

function previous() {
  if (hasPrevious.value) {
    uiStore.previousImage()
    resetZoom()
  }
}

function next() {
  if (hasNext.value) {
    uiStore.nextImage()
    resetZoom()
  }
}

function zoomIn() {
  scale.value = Math.min(scale.value + 0.25, 5)
}

function zoomOut() {
  scale.value = Math.max(scale.value - 0.25, 0.25)
}

function resetZoom() {
  scale.value = 1
}

function handleWheel(event: WheelEvent) {
  if (event.deltaY < 0) {
    zoomIn()
  } else {
    zoomOut()
  }
}

function handleImageLoad() {
  // 图片加载完成
}

async function downloadImage() {
  if (!currentImage.value) return

  try {
    await imageApi.download(currentImage.value.id, currentImage.value.original_name)
  } catch (error) {
    console.error('Download failed:', error)
  }
}

async function deleteImage() {
  if (!currentImage.value) return

  if (!confirm(`确定要删除 "${currentImage.value.original_name}" 吗？`)) {
    return
  }

  try {
    await imageStore.deleteImage(currentImage.value.id)
    close()
  } catch (error) {
    console.error('Delete failed:', error)
    alert('删除失败')
  }
}

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

// 键盘快捷键
function handleKeydown(event: KeyboardEvent) {
  if (!uiStore.imageViewerOpen) return

  switch (event.key) {
    case 'Escape':
      close()
      break
    case 'ArrowLeft':
      previous()
      break
    case 'ArrowRight':
      next()
      break
    case '+':
    case '=':
      zoomIn()
      break
    case '-':
      zoomOut()
      break
    case '0':
      resetZoom()
      break
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>
