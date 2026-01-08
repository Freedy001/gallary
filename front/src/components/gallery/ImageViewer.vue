<template>
  <Teleport to="body">
    <Transition name="fade">
      <div v-if="isVisible"
           class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 liquid-glass-container">
        <div class="absolute inset-0 liquid-glass-backdrop"></div>

        <!-- 关闭按钮 -->
        <button
            @click="close"
            class="absolute right-4 top-4 z-50 rounded-full bg-black/50 p-2 text-white transition-colors hover:bg-black/70 glass-control active:scale-95"
        >
          <XMarkIcon class="h-6 w-6"/>
        </button>

        <!-- 上一张 -->
        <button
            v-if="hasPrevious"
            @click="previous"
            class="absolute left-2 sm:left-4 top-1/2 z-40 -translate-y-1/2 rounded-full bg-black/50 p-2 sm:p-3 text-white transition-colors hover:bg-black/70 glass-control active:scale-95 hidden sm:block"
        >
          <ChevronLeftIcon class="h-5 w-5 sm:h-6 sm:w-6"/>
        </button>

        <!-- 下一张 -->
        <button
            v-if="hasNext"
            @click="next"
            class="absolute right-2 sm:right-4 top-1/2 z-40 -translate-y-1/2 rounded-full bg-black/50 p-2 sm:p-3 text-white transition-colors hover:bg-black/70 glass-control active:scale-95 hidden sm:block"
        >
          <ChevronRightIcon class="h-5 w-5 sm:h-6 sm:w-6"/>
        </button>

        <!-- 图片容器 -->
        <div
            ref="imageContainerRef"
            class="relative flex h-full w-full items-center justify-center overflow-hidden z-0 touch-none"
            @wheel.prevent="handleWheel"
            @mousedown="handleMouseDown"
            @mousemove="handleMouseMove"
            @mouseup="handleMouseUp"
            @mouseleave="handleMouseUp"
            @touchstart="handleTouchStart"
            @touchmove="handleTouchMove"
            @touchend="handleTouchEnd"
            @click.self="close"
        >
          <!-- 加载动画 -->
          <div v-if="!currentImage" class="flex flex-col items-center justify-center gap-4">
            <div class="h-12 w-12 rounded-full border-4 border-white/20 border-t-white animate-spin"></div>
            <span class="text-white/70 text-sm">加载中...</span>
          </div>

          <!-- 图片内容 -->
          <Transition v-else :name="slideDirection">
            <div :key="currentImage.id" class="absolute inset-0 flex items-center justify-center w-full h-full">
              <!-- 占位层：缩略图 + Loading -->
              <div v-if="!isImageLoaded" class="absolute inset-0 flex items-center justify-center z-0 pointer-events-none overflow-hidden">
                <img
                    v-if="currentImage.thumbnail_url"
                    :src="currentImage.thumbnail_url"
                    class="w-full h-full object-contain animate-breathe opacity-80 will-change-transform"
                    alt="thumbnail"
                />
                <div class="absolute z-10 flex flex-col items-center justify-center">
                   <div class="relative flex items-center justify-center">
                      <svg class="w-16 h-16 transform -rotate-90">
                        <circle cx="32" cy="32" fill="none" r="28" stroke="rgba(255,255,255,0.2)" stroke-width="4"/>
                        <circle :stroke-dasharray="175.93" class="transition-all duration-300 ease-out" cx="32" cy="32" fill="none" r="28"
                          stroke="rgba(255,255,255,0.9)" stroke-linecap="round"
                          :stroke-dashoffset="175.93 * (1 - downloadProgress / 100)"
                          stroke-width="4"/>
                      </svg>
                      <span class="absolute text-white text-sm font-medium">{{ downloadProgress }}%</span>
                   </div>
                   <span class="mt-3 text-white/70 text-xs tracking-wide">{{ formatFileSize(currentImage.file_size) }}</span>
                </div>
              </div>

              <!-- 原图 -->
              <img
                  v-if="displayImageUrl"
                  ref="mainImageRef"
                  :src="displayImageUrl"
                  @load="onImageLoad"
                  @error="onImageError"
                  :alt="currentImage.original_name"
                  class="max-h-full max-w-full object-contain transition-all duration-300 shadow-2xl z-10 relative"
                  :class="{
                    'cursor-grab': !originScale() && !isDragging,
                    'cursor-grabbing': isDragging,
                    'opacity-0': !isImageLoaded,
                    'opacity-100': isImageLoaded
                  }"
                  :style="{
                  transform: `translate(${translate.x}px, ${translate.y}px) scale(${scale})`,
                  transition: (isDragging || isSwiping) ? 'none' : 'transform 200ms, opacity 300ms',
                  '-webkit-touch-callout': isWeChat ? 'default' : 'none'
                }"
                  style="user-select: none; -webkit-user-select: none;"
                  draggable="false"
              />
            </div>
          </Transition>
        </div>

        <!-- Details Toggle Button -->
        <Transition name="fade">
          <button
              v-if="!showDetails"
              @click="showDetails = true"
              class="absolute bottom-6 right-6 z-30 rounded-full bg-black/50 p-3 text-white transition-colors hover:bg-black/70 glass-control"
              title="显示详情 (I)"
          >
            <InformationCircleIcon class="h-6 w-6"/>
          </button>
        </Transition>

        <!-- 底部工具栏 -->
        <Transition name="slide-up">
          <div
              v-if="showDetails"
              class="absolute bottom-0 left-0 right-0 p-3 sm:p-6 z-40"
              @click.stop
          >
            <LiquidGlassCard
                class="mx-auto w-full sm:max-w-4xl menu-item relative"
                :content-class="'p-3 sm:p-4 pb-3.5'"
                :target-element="mainImageRef"
                :target-image="imageUrl"
            >
              <button
                  @click="showDetails = false"
                  class="absolute top-2 right-2 sm:right-5 p-2 text-white/70 hover:text-white hover:bg-white/10 rounded-full transition-colors"
                  title="隐藏详情 (I)"
              >
                <ChevronDownIcon class="h-5 w-5"/>
              </button>

              <!-- 微信提示 -->
              <div v-if="isWeChat" class="mb-4 text-center">
                <p class="text-white/90 text-sm font-medium bg-white/10 py-2 px-4 rounded-lg inline-block backdrop-blur-sm">
                  长按图片可保存或发送给朋友
                </p>
              </div>

              <!-- 文件信息 -->
              <div class="mb-3 text-white pr-8">
                <template v-if="!currentImage">
                  <div class="h-6 sm:h-7 w-32 sm:w-48 bg-white/20 rounded animate-pulse"></div>
                  <div class="mt-2 flex gap-4">
                    <div class="h-4 sm:h-5 w-24 sm:w-32 bg-white/10 rounded animate-pulse"></div>
                  </div>
                </template>
                <template v-else>
                  <h3 class="text-base sm:text-lg font-semibold text-shadow truncate">{{ currentImage.original_name }}</h3>
                  <div class="mt-1.5 sm:mt-2 flex flex-wrap items-center gap-2 sm:gap-4 text-xs sm:text-sm text-gray-100 text-shadow-sm">
                    <span v-if="currentImage.taken_at">{{ formatDate(currentImage.taken_at) }}</span>
                    <span v-if="currentImage.camera_model" class="hidden sm:inline">{{ currentImage.camera_model }}</span>
                    <div v-if="currentImage.aperture || currentImage.shutter_speed || currentImage.iso"
                         class="hidden sm:flex gap-3 border-l border-white/30 pl-3">
                      <span v-if="currentImage.aperture">{{ currentImage.aperture }}</span>
                      <span v-if="currentImage.shutter_speed">{{ currentImage.shutter_speed }}s</span>
                      <span v-if="currentImage.iso">ISO{{ currentImage.iso }}</span>
                      <span v-if="currentImage.focal_length">{{ currentImage.focal_length }}</span>
                    </div>
                    <span class="border-l border-white/30 pl-3">{{ currentImage.width }} × {{ currentImage.height }}</span>
                    <span>{{ formatFileSize(currentImage.file_size) }}</span>
                  </div>
                </template>
              </div>

              <!-- 操作按钮 -->
              <div class="flex items-center justify-between gap-2 overflow-x-auto no-scrollbar" style="user-select: none">
                <!-- 缩放控制组 -->
                <div class="flex items-center bg-white/5 rounded-lg border border-white/10 p-0.5 shrink-0">
                  <button class="rounded-md p-1.5 sm:px-3 sm:py-2 text-sm text-white hover:bg-white/10 transition-all" @click="zoomOut(0.25)">
                    <MinusIcon class="h-4 w-4"/>
                  </button>
                  <span class="px-2 text-xs sm:text-sm text-white font-medium min-w-[3rem] text-center">{{ Math.round(scale * 100) }}%</span>
                  <button class="rounded-md p-1.5 sm:px-3 sm:py-2 text-sm text-white hover:bg-white/10 transition-all" @click="zoomIn(0.25)">
                    <PlusIcon class="h-4 w-4"/>
                  </button>
                </div>

                <button
                    @click="resetZoom"
                    class="hidden sm:block rounded-lg bg-white/10 px-3 py-1.5 text-sm text-white hover:bg-white/20 backdrop-blur-md transition-all border border-white/10 shrink-0"
                >
                  重置
                </button>

                <div class="flex-1 flex justify-center px-2 min-w-0">
                  <button
                      @click="showThumbnails = !showThumbnails"
                      class="flex items-center justify-center gap-2 text-sm text-white/70 hover:text-white transition-colors py-1 w-full sm:w-auto"
                  >
                    <span class="hidden sm:inline" v-if="showThumbnails">隐藏预览</span>
                    <span v-else class="hidden sm:inline">显示预览 ({{ props.images.length }})</span>
                    <ChevronUpIcon v-if="showThumbnails" class="h-4 w-4"/>
                    <ChevronDownIcon v-else class="h-4 w-4"/>
                  </button>
                </div>

                <div class="flex items-center gap-2 shrink-0">
                  <button
                      v-if="!isWeChat"
                      @click="downloadImage"
                      class="flex items-center gap-2 rounded-lg bg-white/10 p-2 sm:px-4 sm:py-2 text-sm text-white hover:bg-white/20 backdrop-blur-md transition-all border border-white/10"
                      title="下载"
                  >
                    <ArrowDownTrayIcon class="h-4 w-4"/>
                    <span class="hidden sm:inline">下载</span>
                  </button>

                  <button
                      @click="deleteImage"
                      class="flex items-center gap-2 rounded-lg bg-red-600/80 p-2 sm:px-4 sm:py-2 text-sm text-white hover:bg-red-600 backdrop-blur-md transition-all shadow-lg"
                      title="删除"
                  >
                    <TrashIcon class="h-4 w-4"/>
                    <span class="hidden sm:inline">删除</span>
                  </button>
                </div>
              </div>

              <!-- 缩略图列表 -->
              <div>
                <Transition name="thumbnail-slide">
                  <div
                      v-if="showThumbnails"
                      ref="thumbnailsRef"
                      class="flex gap-1 sm:gap-2 overflow-x-auto px-1 mt-3 no-scrollbar pb-1"
                      @wheel.stop
                  >
                    <div
                        v-for="(img, idx) in props.images"
                        :key="img?.id || idx"
                        class="relative h-10 w-10 flex-shrink-0 cursor-pointer overflow-hidden rounded-lg border-2 transition-all bg-gray-800/50"
                        :class="idx === index ? 'border-blue-500 opacity-100 ring-2 ring-blue-500/50' : 'border-transparent opacity-60 hover:opacity-80'"
                        @click="changeIndex(idx)"
                    >
                      <CachedImage
                          v-if="img"
                          :src="img.thumbnail_url || img.url"
                          img-class="h-full w-full object-cover"
                          :alt="img.original_name"
                      />
                      <div v-else class="h-full w-full flex items-center justify-center text-xs text-gray-500">
                        加载中...
                      </div>
                    </div>
                  </div>
                </Transition>
              </div>
            </LiquidGlassCard>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import {computed, onMounted, onUnmounted, ref, watch} from 'vue'
import LiquidGlassCard from '@/components/widgets/common/LiquidGlassCard.vue'
import CachedImage from '@/components/widgets/common/CachedImage.vue'
import {
  ArrowDownTrayIcon,
  ChevronDownIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronUpIcon,
  InformationCircleIcon,
  MinusIcon,
  PlusIcon,
  TrashIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import {useDialogStore} from "@/stores/dialog"
import {imageApi} from '@/api/image'
import type {Image} from '@/types'

// Props
const props = defineProps<{
  images: (Image | null)[]
  index: number
}>()

// Emits
const emit = defineEmits<{
  (e: 'update:index', value: number): void
  (e: 'delete', id: number): void
}>()

const dialogStore = useDialogStore()

const imageContainerRef = ref<HTMLElement>()
const scale = ref(1)
const translate = ref({x: 0, y: 0})
const isDragging = ref(false)
const dragStart = ref({x: 0, y: 0})
const touchStart = ref({x: 0, y: 0})
const initialTouchDistance = ref(0)
const initialTouchScale = ref(1)
const isSwiping = ref(false)
const isImageLoaded = ref(false)
const downloadProgress = ref(0)
const blobUrl = ref<string>('')
const abortController = ref<AbortController | null>(null)

const slideDirection = ref<'slide-left' | 'slide-right'>('slide-left')
const showDetails = ref(true)
const showThumbnails = ref(true)
const thumbnailsRef = ref<HTMLElement>()
const isWeChat = ref(false)

const index = computed({
  get: () => props.index,
  set: (val) => emit('update:index', val)
})

const isVisible = computed(() => index.value !== -1)
const currentImage = computed(() => props.images[index.value] || null)

watch(() => currentImage.value?.id, async (newId, oldId) => {
  if (newId === oldId) return

  if (abortController.value) {
    abortController.value.abort()
  }

  if (blobUrl.value) {
    URL.revokeObjectURL(blobUrl.value)
    blobUrl.value = ''
  }

  isImageLoaded.value = false
  downloadProgress.value = 0

  if (currentImage.value?.url) {
    await downloadImageWithProgress(currentImage.value.url, currentImage.value.file_size)
  }
})

async function downloadImageWithProgress(url: string, expectedSize?: number) {
  abortController.value = new AbortController()

  try {
    const response = await fetch(url, { signal: abortController.value.signal })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const contentLength = response.headers.get('Content-Length')
    const totalSize = contentLength ? parseInt(contentLength, 10) : (expectedSize || 0)

    if (!response.body) {
      const blob = await response.blob()
      blobUrl.value = URL.createObjectURL(blob)
      downloadProgress.value = 100
      return
    }

    const reader = response.body.getReader()
    const chunks: Uint8Array[] = []
    let receivedLength = 0

    while (true) {
      const {done, value} = await reader.read()
      if (done) break

      chunks.push(value)
      receivedLength += value.length

      if (totalSize > 0) {
        downloadProgress.value = Math.min(Math.round((receivedLength / totalSize) * 100), 99)
      } else {
        downloadProgress.value = Math.min(Math.round(receivedLength / 1024 / 1024 * 10), 99)
      }
    }

    const blob = new Blob(chunks as BlobPart[])
    blobUrl.value = URL.createObjectURL(blob)
    downloadProgress.value = 100

  } catch (error) {
    if ((error as Error).name === 'AbortError') return
    console.error('图片下载失败:', error)
    blobUrl.value = url
    downloadProgress.value = 100
  }
}

function onImageLoad(event: Event) {
  const img = event.target as HTMLImageElement
  if (img && img.complete && img.naturalWidth > 0) {
    isImageLoaded.value = true
  }
}

function onImageError() {
  isImageLoaded.value = true
  console.error('图片加载失败:', displayImageUrl.value)
}

const displayImageUrl = computed(() => blobUrl.value || '')
const imageUrl = computed(() => currentImage.value?.url || '')

const hasPrevious = computed(() => index.value > 0)
const hasNext = computed(() => index.value < props.images.length - 1)

function close() {
  index.value = -1
  resetZoom()
}

function previous() {
  if (hasPrevious.value) {
    slideDirection.value = 'slide-right'
    index.value--
    resetZoom()
  }
}

function next() {
  if (hasNext.value) {
    slideDirection.value = 'slide-left'
    index.value++
    resetZoom()
  }
}

function changeIndex(newIndex: number) {
  if (newIndex >= 0 && newIndex < props.images.length) {
    if (newIndex > index.value) {
      slideDirection.value = 'slide-left'
    } else {
      slideDirection.value = 'slide-right'
    }
    index.value = newIndex
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

function getDistance(touches: TouchList) {
  return touches[0] && touches[1] ? Math.hypot(
      touches[0].clientX - touches[1].clientX,
      touches[0].clientY - touches[1].clientY
  ) : null
}

function handleTouchStart(e: TouchEvent) {
  if (e.touches.length === 1 && e.touches[0]) {
    const touch = e.touches[0]
    touchStart.value = {x: touch.clientX, y: touch.clientY}

    if (scale.value > 1) {
      isDragging.value = true
      dragStart.value = {
        x: touch.clientX - translate.value.x,
        y: touch.clientY - translate.value.y
      }
    } else {
      isSwiping.value = true
    }
  } else if (e.touches.length === 2) {
    isDragging.value = false
    isSwiping.value = false
    initialTouchDistance.value = getDistance(e.touches) ?? 0
    initialTouchScale.value = scale.value
  }
}

function handleTouchMove(e: TouchEvent) {
  if (e.cancelable) {
    e.preventDefault()
  }

  if (e.touches.length === 1 && e.touches[0]) {
    const touch = e.touches[0]

    if (scale.value > 1 && isDragging.value) {
      translate.value = {
        x: touch.clientX - dragStart.value.x,
        y: touch.clientY - dragStart.value.y
      }
    } else if (scale.value === 1 && isSwiping.value) {
      const deltaX = touch.clientX - touchStart.value.x
      translate.value = {x: deltaX, y: 0}
    }
  } else if (e.touches.length === 2) {
    const currentDistance = getDistance(e.touches)
    if (initialTouchDistance.value > 0 && currentDistance) {
      const ratio = currentDistance / initialTouchDistance.value
      scale.value = Math.min(Math.max(initialTouchScale.value * ratio, 0.5), 5)
    }
  }
}

function handleTouchEnd(e: TouchEvent) {
  if (e.touches.length === 0) {
    if (scale.value <= 1 && isSwiping.value) {
      const deltaX = translate.value.x
      const threshold = 50

      if (Math.abs(deltaX) > threshold) {
        if (deltaX > 0) {
          previous()
        } else {
          next()
        }
      } else {
        resetZoom()
      }
    }
    isDragging.value = false
    isSwiping.value = false
  } else if (e.touches.length === 1 && e.touches[0]) {
    const touch = e.touches[0]
    dragStart.value = {
      x: touch.clientX - translate.value.x,
      y: touch.clientY - translate.value.y
    }
    isDragging.value = scale.value > 1
    isSwiping.value = false
  }
}

function originScale(): boolean {
  return scale.value == 1 && translate.value.x === 0 && translate.value.y === 0
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

  const confirmed = await dialogStore.confirm({
    title: '确认删除',
    message: `确定要删除 "${currentImage.value.original_name}" 吗？`,
    type: 'warning',
    confirmText: '删除'
  })

  if (!confirmed) return

  try {
    await imageApi.deleteBatch([currentImage.value.id])
    emit('delete', currentImage.value.id)
  } catch (error) {
    console.error('Delete failed:', error)
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

function handleKeydown(event: KeyboardEvent) {
  if (!isVisible.value) return

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
    case 'i':
    case 'I':
      showDetails.value = !showDetails.value
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

watch(index, async (newIndex) => {
  if (showThumbnails.value && thumbnailsRef.value) {
    const container = thumbnailsRef.value
    const children = container.children
    if (children[newIndex]) {
      const element = children[newIndex] as HTMLElement
      const containerCenter = container.clientWidth / 2
      const elementCenter = element.offsetLeft + element.clientWidth / 2
      container.scrollTo({
        left: elementCenter - containerCenter,
        behavior: 'smooth'
      })
    }
  }
})

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  const ua = navigator.userAgent.toLowerCase()
  isWeChat.value = ua.includes('micromessenger')

  if (currentImage.value?.url) {
    downloadImageWithProgress(currentImage.value.url, currentImage.value.file_size)
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)

  if (abortController.value) {
    abortController.value.abort()
  }
  if (blobUrl.value) {
    URL.revokeObjectURL(blobUrl.value)
  }
})
</script>

<style scoped>
.slide-left-enter-active,
.slide-left-leave-active,
.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.25s ease-in-out;
}

.slide-left-enter-from { transform: translateX(100%); }
.slide-left-leave-to { transform: translateX(-100%); }
.slide-right-enter-from { transform: translateX(-100%); }
.slide-right-leave-to { transform: translateX(100%); }

.liquid-glass-container { perspective: 1000px; }

.liquid-glass-backdrop {
  background: radial-gradient(circle at center, rgba(40, 40, 50, 0.4) 0%, rgba(10, 10, 15, 0.08) 100%);
  backdrop-filter: blur(10px);
}

.menu-item { transition: all 0.4s cubic-bezier(0.25, 0.8, 0.25, 1); }

.text-shadow { text-shadow: 0 2px 4px rgba(0, 0, 0, 0.5); }
.text-shadow-sm { text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5); }

.glass-control {
  background: rgba(80, 80, 80, 0.3);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.glass-control:hover {
  background: rgba(80, 80, 80, 0.5);
  border-color: rgba(255, 255, 255, 0.3);
  transform: scale(1.05);
}

.glass-control.top-4:hover { transform: scale(1.05); }

.slide-up-enter-active, .slide-up-leave-active { transition: all 0.3s ease; }
.slide-up-enter-from, .slide-up-leave-to { transform: translateY(100%); opacity: 0; }

.thumbnail-slide-enter-active, .thumbnail-slide-leave-active {
  transition: all 0.3s ease;
  max-height: 100px;
  opacity: 1;
}
.thumbnail-slide-enter-from, .thumbnail-slide-leave-to {
  max-height: 0;
  margin-top: 0;
  opacity: 0;
}

@keyframes breathe {
  0%, 100% { filter: brightness(0.8) blur(8px); transform: scale(1.02); }
  50% { filter: brightness(1.1) blur(12px); transform: scale(1.03); }
}

.animate-breathe { animation: breathe 2s ease-in-out infinite; }
</style>
