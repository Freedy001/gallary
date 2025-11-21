<template>
  <Teleport to="body">
    <Transition name="fade">
      <!--
        Change: Changed base container to handle the liquid glass context
        Instead of simple bg-black, we'll use our custom glass style classes
      -->
      <div
          v-if="uiStore.imageViewerOpen && currentImage"
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 liquid-glass-container"
      >
        <!-- Layer 1: Glass Distortion Background (The overlay itself acts as the glass) -->
        <div class="absolute inset-0 liquid-glass-backdrop"></div>

        <!-- 关闭按钮 -->
        <button
            @click="close"
            class="absolute right-4 top-4 z-10 rounded-full bg-black/50 p-2 text-white transition-colors hover:bg-black/70 glass-control"
        >
          <XMarkIcon class="h-6 w-6"/>
        </button>

        <!-- 上一张 -->
        <button
            v-if="hasPrevious"
            @click="previous"
            class="absolute left-4 top-1/2 z-10 -translate-y-1/2 rounded-full bg-black/50 p-3 text-white transition-colors hover:bg-black/70 glass-control"
        >
          <ChevronLeftIcon class="h-6 w-6"/>
        </button>

        <!-- 下一张 -->
        <button
            v-if="hasNext"
            @click="next"
            class="absolute right-4 top-1/2 z-10 -translate-y-1/2 rounded-full bg-black/50 p-3 text-white transition-colors hover:bg-black/70 glass-control"
        >
          <ChevronRightIcon class="h-6 w-6"/>
        </button>

        <!-- 图片容器 -->
        <div
            ref="imageContainerRef"
            class="relative flex h-full w-full items-center justify-center overflow-hidden z-0"
            @wheel.prevent="handleWheel"
            @mousedown="handleMouseDown"
            @mousemove="handleMouseMove"
            @mouseup="handleMouseUp"
            @mouseleave="handleMouseUp"
            @click.self="close"
        >
          <Transition :name="slideDirection">
            <div :key="currentImage.id" class="absolute inset-0 flex items-center justify-center">
              <img
                  ref="mainImageRef"
                  :src="imageUrl"
                  :alt="currentImage.original_name"
                  class="max-h-full max-w-full object-contain transition-transform duration-200 shadow-2xl"
                  :class="{ 'cursor-grab': !originScale() && !isDragging, 'cursor-grabbing': isDragging }"
                  :style="{
                  transform: `translate(${translate.x}px, ${translate.y}px) scale(${scale})`,
                  transition: isDragging ? 'none' : 'transform 200ms'
                }"
                  style="user-select: none"
                  draggable="false"
                  @load="handleImageLoad"
              />
            </div>
          </Transition>
        </div>

        <!-- 底部工具栏 (Liquid Glass Card Style) -->
        <div
            class="absolute bottom-0 left-0 right-0 p-6 z-20"
            @click.stop
        >
          <LiquidGlassCard
              class="mx-auto max-w-4xl menu-item"
              :target-element="mainImageRef"
              :target-image="imageUrl"
          >
              <!-- 文件信息 -->
              <div class="mb-4 text-white">
                <h3 class="text-lg font-semibold text-shadow">{{ currentImage.original_name }}</h3>
                <div class="mt-2 flex flex-wrap gap-4 text-sm text-gray-100 text-shadow-sm">
                  <span v-if="currentImage.taken_at">{{ formatDate(currentImage.taken_at) }}</span>
                  <span v-if="currentImage.camera_model">{{ currentImage.camera_model }}</span>
                  <!-- EXIF 信息 -->
                  <div v-if="currentImage.aperture || currentImage.shutter_speed || currentImage.iso"
                       class="flex gap-3 border-l border-white/30 pl-3">
                    <span v-if="currentImage.aperture">f/{{ currentImage.aperture }}</span>
                    <span v-if="currentImage.shutter_speed">{{ currentImage.shutter_speed }}s</span>
                    <span v-if="currentImage.iso">ISO{{ currentImage.iso }}</span>
                  </div>
                  <span class="border-l border-white/30 pl-3">{{ currentImage.width }} × {{ currentImage.height }}</span>
                  <span>{{ formatFileSize(currentImage.file_size) }}</span>
                </div>
              </div>

              <!-- 操作按钮 -->
              <div class="flex items-center gap-2" style="user-select: none">
                <button
                    @click="zoomOut(0.25)"
                    class="rounded-lg bg-white/10 px-3 py-2 text-sm text-white hover:bg-white/20 backdrop-blur-md transition-all border border-white/10"
                >
                  <MinusIcon class="h-4 w-4"/>
                </button>
                <span class="px-3 text-sm text-white font-medium text-shadow-sm">{{ Math.round(scale * 100) }}%</span>
                <button
                    @click="zoomIn(0.25)"
                    class="rounded-lg bg-white/10 px-3 py-2 text-sm text-white hover:bg-white/20 backdrop-blur-md transition-all border border-white/10"
                >
                  <PlusIcon class="h-4 w-4"/>
                </button>
                <button
                    @click="resetZoom"
                    class="rounded-lg bg-white/10 px-3 py-2 text-sm text-white hover:bg-white/20 backdrop-blur-md transition-all border border-white/10"
                >
                  重置
                </button>

                <div class="flex-1"/>

                <button
                    @click="downloadImage"
                    class="flex items-center gap-2 rounded-lg bg-white/10 px-4 py-2 text-sm text-white hover:bg-white/20 backdrop-blur-md transition-all border border-white/10"
                >
                  <ArrowDownTrayIcon class="h-4 w-4"/>
                  下载
                </button>

                <button
                    @click="deleteImage"
                    class="flex items-center gap-2 rounded-lg bg-red-600/80 px-4 py-2 text-sm text-white hover:bg-red-600 backdrop-blur-md transition-all shadow-lg"
                >
                  <TrashIcon class="h-4 w-4"/>
                  删除
                </button>
              </div>
          </LiquidGlassCard>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import {computed, ref, onMounted, onUnmounted} from 'vue'
import {useUIStore} from '@/stores/ui'
import {useImageStore} from '@/stores/image'
import {imageApi} from '@/api/image'
import LiquidGlassCard from '@/components/common/LiquidGlassCard.vue'
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
const translate = ref({x: 0, y: 0})
const isDragging = ref(false)
const dragStart = ref({x: 0, y: 0})
const slideDirection = ref<'slide-left' | 'slide-right'>('slide-left')

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
    slideDirection.value = 'slide-right'
    uiStore.previousImage()
    resetZoom()
  }
}

function next() {
  if (hasNext.value) {
    slideDirection.value = 'slide-left'
    uiStore.nextImage()
    resetZoom()
  }
}

function zoomIn(step: number | undefined) {
  scale.value = Math.min(scale.value + (step || 0.25), 5)
}

function zoomOut(step: number | undefined) {
  scale.value = Math.max(scale.value - (step || 0.25), 0.25)
}

function resetZoom() {
  scale.value = 1
  translate.value = {x: 0, y: 0}
}

function handleWheel(event: WheelEvent) {
  if (event.deltaY < 0) {
    zoomIn(0.1)
  } else {
    zoomOut(0.1)
  }
}

function handleMouseDown(e: MouseEvent) {
  if (originScale()) return
  e.preventDefault()
  isDragging.value = true
  dragStart.value = {x: e.clientX - translate.value.x, y: e.clientY - translate.value.y}
}

function handleMouseMove(e: MouseEvent) {
  if (!isDragging.value) return
  e.preventDefault()
  translate.value = {
    x: e.clientX - dragStart.value.x,
    y: e.clientY - dragStart.value.y
  }
}

function handleMouseUp() {
  isDragging.value = false
}

function originScale(): boolean {
  return scale.value == 1 && translate.value.x === 0 && translate.value.y === 0;
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
      zoomIn(0.25)
      break
    case '-':
      zoomOut(0.25)
      break
    case '0':
      resetZoom()
      break
  }
}

const mainImageRef = ref<HTMLImageElement>()

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
/* 基础动画 */
.slide-left-enter-active,
.slide-left-leave-active,
.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.25s ease-in-out;
}

.slide-left-enter-from {
  transform: translateX(100%);
}

.slide-left-leave-to {
  transform: translateX(-100%);
}

.slide-right-enter-from {
  transform: translateX(-100%);
}

.slide-right-leave-to {
  transform: translateX(100%);
}

/* --- Liquid Glass Style --- */

/* 容器背景 */
.liquid-glass-container {
  perspective: 1000px;
}

/* 玻璃背景层 */
.liquid-glass-backdrop {
  background: radial-gradient(circle at center, rgba(40, 40, 50, 0.4) 0%, rgba(10, 10, 15, 0.08) 100%);
  backdrop-filter: blur(10px);
}

/* 鼠标悬停效果 */
.menu-item {
  transition: all 0.4s cubic-bezier(0.25, 0.8, 0.25, 1);
}

/* 文字阴影增强可读性 */
.text-shadow {
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.5);
}
.text-shadow-sm {
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}

/* 控制按钮玻璃风格 */
.glass-control {
  background: rgba(80, 80, 80, 0.3);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}
.glass-control:hover {
  background: rgba(80, 80, 80, 0.5);
  border-color: rgba(255, 255, 255, 0.3);
  transform: scale(1.05); /* Keep vertical center */
}
/* Top close button needs simpler hover transform */
.glass-control.top-4:hover {
  transform: scale(1.05);
}

</style>
