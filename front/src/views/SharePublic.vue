<template>
  <div class="h-screen overflow-hidden bg-transparent">
    <!-- 验证加载中 -->
    <div v-if="imageStore.loading" class="flex h-full items-center justify-center">
      <div class="text-center">
        <div class="h-10 w-10 animate-spin rounded-full border-4 border-blue-500 border-t-transparent mx-auto"></div>
        <p class="mt-4 text-white/60 text-sm">加载中...</p>
      </div>
    </div>

    <!-- 密码验证 -->
    <div v-else-if="needPassword" class="flex h-full items-center justify-center px-4">
      <div class="liquid-glass-card w-full max-w-md">
        <div class="glass-highlight"></div>
        <div class="relative z-10 p-8">
          <div class="text-center">
            <div
                class="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-white/10 backdrop-blur-sm">
              <LockClosedIcon class="h-8 w-8 text-white"/>
            </div>
            <h2 class="text-2xl font-bold text-white">需要访问密码</h2>
            <p class="mt-3 text-white/60">
              {{ shareInfo?.title || '该分享' }} 受密码保护
            </p>
          </div>

          <form @submit.prevent="handleVerify" class="mt-8 space-y-4">
            <div>
              <label class="sr-only">密码</label>
              <input
                  v-model="password"
                  type="password"
                  required
                  placeholder="请输入访问密码"
                  class="glass-input text-center"
              />
            </div>
            <p v-if="error" class="text-sm text-red-400 text-center">{{ error }}</p>
            <button
                type="submit"
                :disabled="verifying"
                class="glass-button-primary w-full"
            >
              <span v-if="verifying" class="flex items-center justify-center gap-2">
                <svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none"/>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
                </svg>
                验证中...
              </span>
              <span v-else>查看分享</span>
            </button>
          </form>
        </div>
      </div>
    </div>

    <!-- 分享内容 -->
    <div v-else-if="shareInfo" class="h-full flex flex-col">
      <!-- 头部 -->
      <header class="flex-shrink-0 border-b border-white/10 backdrop-blur-xl bg-black/30">
        <div class="mx-auto max-w-7xl px-4 py-5 sm:px-6 lg:px-8">
          <!-- 选择模式下的头部 -->
          <div v-if="uiStore.isSelectionMode" class="flex items-center justify-between">
            <div class="flex items-center gap-4">
              <span class="text-lg font-medium text-white">已选择 {{ imageStore.selectedCount }} 项</span>
              <button
                  @click="handleSelectAll"
                  class="text-sm text-blue-400 hover:text-blue-300"
              >
                {{ isAllSelected ? '取消全选' : '全选' }}
              </button>
            </div>
            <div class="flex items-center gap-3">
              <button
                  v-if="imageStore.selectedCount > 0"
                  @click="downloadSelected"
                  :disabled="downloading"
                  class="glass-button-primary flex items-center gap-2"
              >
                <ArrowDownTrayIcon class="h-4 w-4"/>
                <span>{{ downloading ? '下载中...' : `下载 (${imageStore.selectedCount})` }}</span>
              </button>
              <button
                  @click="exitSelectionMode"
                  class="glass-button-secondary"
              >
                完成
              </button>
            </div>
          </div>

          <!-- 正常模式下的头部 -->
          <div v-else class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
            <div>
              <h1 class="text-2xl font-bold text-white">{{ shareInfo.title || '未命名分享' }}</h1>
              <p v-if="shareInfo.description" class="mt-2 text-white/70">
                {{ shareInfo.description }}
              </p>
              <div class="mt-2 flex items-center gap-4 text-sm text-white/50">
                <span class="flex items-center gap-1.5">
                  <PhotoIcon class="h-4 w-4"/>
                  {{ imageStore.total }} 张照片
                </span>
                <span class="flex items-center gap-1.5">
                  <ClockIcon class="h-4 w-4"/>
                  {{ formatDate(shareInfo.created_at) }} 分享
                </span>
              </div>
            </div>
            <div class="flex items-center gap-3">
              <button
                  @click="enterSelectionMode"
                  class="glass-button-secondary"
              >
                选择
              </button>
            </div>
          </div>
        </div>
      </header>

      <!-- 图片网格 -->
      <main ref="scrollContainerRef" class="flex-1 overflow-y-auto px-4 py-8 sm:px-6 lg:px-8">
        <div class="mx-auto max-w-7xl">
          <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
            <div
                v-for="(image, index) in imageStore.images"
                :key="image?.id || index"
                :ref="(el) => setItemRef(el as HTMLElement | null, index)"
                :data-index="index"
                class="group relative aspect-square cursor-pointer overflow-hidden rounded-xl bg-white/5 border border-white/10 transition-all hover:border-white/20 hover:scale-[1.02]"
                :class="[
                uiStore.isSelectionMode && image && imageStore.selectedImages.has(image.id)
                  ? 'ring-2 ring-blue-500 border-blue-500'
                  : ''
              ]"
                @click="imageStore.viewerIndex=index"
            >
              <template v-if="image">
                <img
                    :src="imageApi.getImageUrl(image.thumbnail_path || image.storage_path)"
                    :alt="image.original_name"
                    class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                    loading="lazy"
                />
                <!-- 选择模式下的选择指示器 -->
                <div
                    v-if="uiStore.isSelectionMode"
                    class="absolute top-2 right-2 z-10"
                >
                  <div
                      class="flex h-6 w-6 items-center justify-center rounded-full border-2 transition-colors"
                      :class="[
                      imageStore.selectedImages.has(image.id)
                        ? 'border-blue-500 bg-blue-500 text-white'
                        : 'border-white/70 bg-black/40'
                    ]"
                  >
                    <CheckIcon v-if="imageStore.selectedImages.has(image.id)" class="h-4 w-4"/>
                  </div>
                </div>
                <!-- 悬浮遮罩 -->
                <div
                    class="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent transition-opacity"
                    :class="[uiStore.isSelectionMode ? 'opacity-30' : 'opacity-0 group-hover:opacity-100']"
                >
                  <div class="absolute bottom-0 left-0 right-0 p-3">
                    <p class="text-white text-sm truncate">{{ image.original_name }}</p>
                  </div>
                </div>
              </template>
              <template v-else>
                <div class="h-full w-full flex items-center justify-center">
                  <div class="h-6 w-6 animate-spin rounded-full border-2 border-white/30 border-t-white/80"></div>
                </div>
              </template>
            </div>
          </div>

        </div>
      </main>
    </div>

    <!-- 错误状态 -->
    <div v-else class="flex min-h-screen items-center justify-center">
      <div class="liquid-glass-card max-w-md mx-4">
        <div class="glass-highlight"></div>
        <div class="relative z-10 p-8 text-center">
          <div class="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-red-500/10">
            <ExclamationTriangleIcon class="h-8 w-8 text-red-400"/>
          </div>
          <h3 class="text-xl font-semibold text-white">无法访问</h3>
          <p class="mt-3 text-white/60">{{ error || '分享不存在或已过期' }}</p>
        </div>
      </div>
    </div>

    <!-- 复用 ImageViewer 组件 -->
    <ImageViewer/>
  </div>
</template>

<script setup lang="ts">
import {ref, computed, onMounted, onUnmounted, nextTick} from 'vue'
import {useRoute} from 'vue-router'
import {shareApi} from '@/api/share'
import {imageApi} from '@/api/image'
import {useImageStore} from '@/stores/image'
import {useUIStore} from '@/stores/ui'
import ImageViewer from '@/components/gallery/ImageViewer.vue'
import {
  LockClosedIcon,
  ExclamationTriangleIcon,
  PhotoIcon,
  ClockIcon,
  ArrowDownTrayIcon,
  CheckIcon,
} from '@heroicons/vue/24/outline'
import type {Image, SharePublicInfo} from '@/types'

const route = useRoute()
const imageStore = useImageStore()
const uiStore = useUIStore()
const code = route.params.code as string

const verifying = ref(false)
const downloading = ref(false)
const needPassword = ref(false)
const shareInfo = ref<SharePublicInfo | null>(null)
const password = ref('')
const error = ref('')

// 分享页面固定 6 列，计算合适的分页大小
// 6 列 * 5 行/屏 * 2 屏 = 60
const pageSize = 60

// 滚动加载相关
const scrollContainerRef = ref<HTMLElement | null>(null)
const observer = ref<IntersectionObserver | null>(null)
const itemRefs = new Map<number, HTMLElement>()
const loadingPages = new Set<number>()

const isAllSelected = computed(() => {
  const validImages = imageStore.images.filter(img => img !== null)
  return validImages.length > 0 && imageStore.selectedCount === validImages.length
})

// 选择模式相关方法
function enterSelectionMode() {
  uiStore.setSelectionMode(true)
}

function exitSelectionMode() {
  uiStore.setSelectionMode(false)
  imageStore.clearSelection()
}

function handleSelectAll() {
  if (isAllSelected.value) {
    imageStore.clearSelection()
  } else {
    imageStore.images.forEach(img => {
      if (img) imageStore.selectImage(img.id)
    })
  }
}

// 设置元素引用，用于 IntersectionObserver
function setItemRef(el: HTMLElement | null, index: number) {
  if (el) {
    itemRefs.set(index, el)
    if (observer.value) {
      observer.value.observe(el)
    }
  } else {
    itemRefs.delete(index)
  }
}

// 初始化 IntersectionObserver
function initObserver() {
  observer.value = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            const index = Number((entry.target as HTMLElement).dataset.index)
            if (!isNaN(index) && !imageStore.images[index]) {
              // 计算需要加载的页码
              const page = Math.floor(index / pageSize) + 1
              imageStore.fetchImages(page, pageSize)
            }
          }
        })
      },
      {
        root: scrollContainerRef.value,
        rootMargin: '200px 0px', // 提前 200px 加载
        threshold: 0,
      }
  )

  // 观察所有已存在的元素
  itemRefs.forEach((el) => {
    observer.value?.observe(el)
  })
}

async function downloadSelected() {
  if (downloading.value || imageStore.selectedCount === 0) return

  downloading.value = true
  try {
    const selectedIds = Array.from(imageStore.selectedImages)
    for (const id of selectedIds) {
      const image = imageStore.images.find(img => img?.id === id)
      if (image) {
        await imageApi.download(image.id, image.original_name)
        await new Promise(resolve => setTimeout(resolve, 300))
      }
    }
  } catch (err) {
    console.error('Download selected failed:', err)
  } finally {
    downloading.value = false
  }
}


async function handleVerify() {
  verifying.value = true
  error.value = ''
  try {
    await imageStore.fetchImages(1, pageSize)
    needPassword.value = false
  } catch (err: any) {
    error.value = err.message || '验证失败'
    password.value = ''
  } finally {
    verifying.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  })
}

async function checkShare() {
  try {
    const res = await shareApi.getPublicInfo(code)
    shareInfo.value = res.data
    if (res.data.has_password) {
      needPassword.value = true
    } else {
      await imageStore.fetchImages(1, pageSize)
      await nextTick()
      initObserver()
    }
  } catch (err: any) {
    error.value = err.message || '获取分享信息失败'
  }
}

onMounted(() => {
  imageStore.refreshImages(async (page, size) => (await shareApi.getImages(code, password.value, page, size)).data)
  checkShare()
})

onUnmounted(() => {
  // 清理 observer
  if (observer.value) {
    observer.value.disconnect()
    observer.value = null
  }
  itemRefs.clear()
  loadingPages.clear()
  imageStore.clearSelection()
})
</script>

<style scoped>
.liquid-glass-card {
  position: relative;
  border-radius: 24px;
  background: linear-gradient(
      135deg,
      rgba(255, 255, 255, 0.1) 0%,
      rgba(255, 255, 255, 0.05) 100%
  );
  border: 1px solid rgba(255, 255, 255, 0.12);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5),
  inset 0 1px 0 rgba(255, 255, 255, 0.1);
  overflow: hidden;
}

.liquid-glass-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 24px;
  padding: 1px;
  background: linear-gradient(
      135deg,
      rgba(255, 100, 100, 0.12),
      rgba(255, 200, 100, 0.08),
      rgba(100, 255, 200, 0.08),
      rgba(100, 150, 255, 0.12)
  );
  -webkit-mask: linear-gradient(#fff 0 0) content-box,
  linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask: linear-gradient(#fff 0 0) content-box,
  linear-gradient(#fff 0 0);
  mask-composite: exclude;
  pointer-events: none;
  opacity: 0.5;
}

.glass-highlight {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 40%;
  background: linear-gradient(
      to bottom,
      rgba(255, 255, 255, 0.06) 0%,
      rgba(255, 255, 255, 0) 100%
  );
  pointer-events: none;
  z-index: 5;
  border-radius: 24px 24px 0 0;
}

.glass-input {
  width: 100%;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 12px;
  padding: 14px 18px;
  color: white;
  font-size: 15px;
  outline: none;
  transition: all 0.2s ease;
}

.glass-input::placeholder {
  color: rgba(255, 255, 255, 0.4);
}

.glass-input:focus {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.25);
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.05);
}

.glass-button-primary {
  padding: 12px 24px;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 500;
  color: white;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.8), rgba(99, 102, 241, 0.8));
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.2s ease;
  backdrop-filter: blur(8px);
}

.glass-button-primary:hover:not(:disabled) {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.9), rgba(99, 102, 241, 0.9));
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.glass-button-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.glass-button-secondary {
  padding: 12px 24px;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.8);
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.2s ease;
}

.glass-button-secondary:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.12);
  color: white;
}

.glass-button-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
