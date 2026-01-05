<template>
  <div class="h-screen overflow-hidden bg-transparent">
    <!-- 验证加载中 -->
    <div v-if="imageStore.loading" class="flex h-screen items-center justify-center">
      <div class="text-center">
        <div
            class="inline-block h-12 w-12 animate-spin rounded-full border-2 border-white/5 border-t-primary-500 shadow-[0_0_15px_rgba(139,92,246,0.3)]"></div>
        <p class="mt-6 text-sm text-white/40 tracking-[0.2em] uppercase font-medium">Loading</p>
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
      <header class="flex-shrink-0 border-b border-white/10 backdrop-blur-xl bg-black/30 z-20">
        <div class="mx-auto max-w-7xl px-4 py-3 sm:px-6 sm:py-5 lg:px-8">
          <!-- 选择模式下的头部 -->
          <div v-if="uiStore.isSelectionMode" class="flex items-center justify-between min-h-[2.5rem] sm:min-h-[3rem]">
            <div class="flex items-center gap-4">
              <span class="text-base sm:text-lg font-medium text-white">已选择 {{ imageStore.selectedCount }} 项</span>
            </div>
            <div class="flex items-center gap-2 sm:gap-3">
              <button
                  v-if="imageStore.selectedCount > 0"
                  @click="downloadSelected"
                  class="glass-button-primary flex items-center gap-2 !py-1.5 !px-3 !text-sm sm:!py-3 sm:!px-6 sm:!text-base"
              >
                <ArrowDownTrayIcon class="h-4 w-4"/>
                <span class="hidden sm:inline">下载 ({{ imageStore.selectedCount }})</span>
              </button>
              <button
                  @click="exitSelectionMode"
                  class="glass-button-secondary !py-1.5 !px-3 !text-sm sm:!py-3 sm:!px-6 sm:!text-base"
              >
                完成
              </button>
            </div>
          </div>

          <!-- 正常模式下的头部 -->
          <div v-else>
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0 flex-1">
                <h1 class="text-xl sm:text-2xl font-bold text-white truncate leading-tight">{{ shareInfo.title || '未命名分享' }}</h1>
                <div class="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs sm:text-sm text-white/50">
                  <span class="flex items-center gap-1.5">
                    <PhotoIcon class="h-3.5 w-3.5 sm:h-4 sm:w-4"/>
                    {{ imageStore.total }} 张照片
                  </span>
                  <span class="flex items-center gap-1.5">
                    <ClockIcon class="h-3.5 w-3.5 sm:h-4 sm:w-4"/>
                    {{ formatDate(shareInfo.created_at) }} 分享
                  </span>
                </div>
              </div>
              <button
                  @click="enterSelectionMode"
                  class="glass-button-secondary !py-1.5 !px-3 !text-sm sm:!py-3 sm:!px-6 sm:!text-base flex-shrink-0 ml-2"
              >
                选择
              </button>
            </div>

            <p v-if="shareInfo.description" class="mt-3 text-sm sm:text-base text-white/70 leading-relaxed line-clamp-2 sm:line-clamp-none">
              {{ shareInfo.description }}
            </p>
          </div>
        </div>
      </header>

      <!-- 图片网格 -->
      <main
          ref="scrollContainerRef"
          class="flex-1 overflow-y-auto px-2 py-4 sm:px-6 lg:px-8 relative select-none"
          @mousedown="handleMouseDown"
      >
        <div class="mx-auto max-w-7xl">
          <!-- Context Menu -->
          <ContextMenu v-model="contextMenu.visible" :x="contextMenu.x" :y="contextMenu.y">
            <ContextMenuItem v-if="contextMenuTargetIds.length === 1" :icon="EyeIcon" @click="handleContextMenuView">
              查看
            </ContextMenuItem>
            <ContextMenuItem :icon="ArrowDownTrayIcon" @click="handleContextMenuDownload">
              下载 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
            </ContextMenuItem>
            <ContextMenuItem v-if="contextMenuTargetIds.length > 1" :icon="ArchiveBoxArrowDownIcon" @click="handleContextMenuZipDownload">
              打包下载 {{`(${contextMenuTargetIds.length})` }}
            </ContextMenuItem>
          </ContextMenu>

          <!-- Selection Box -->
          <SelectionBox :style="selectionBoxStyle"/>

          <!-- 空状态 -->
          <div v-if="!imageStore.images || imageStore.images.length === 0"
               class="flex min-h-[50vh] flex-col items-center justify-center py-12">
            <div class="text-center relative z-10">
              <div class="relative group mx-auto mb-8">
                <!-- 背景光晕 -->
                <div
                    class="absolute -inset-4 rounded-full bg-primary-500/20 blur-2xl opacity-50 group-hover:opacity-75 transition-all duration-700"></div>

                <!-- 图标容器 -->
                <div
                    class="relative mx-auto flex h-32 w-32 items-center justify-center rounded-full bg-white/5 ring-1 ring-white/10 backdrop-blur-xl shadow-2xl transition-all duration-500 group-hover:scale-105 group-hover:bg-white/10 group-hover:ring-white/20">
                  <SparklesIcon class="h-16 w-16 text-primary-400/90 drop-shadow-[0_0_15px_rgba(139,92,246,0.5)]"/>
                </div>
              </div>

              <h3 class="text-3xl font-bold tracking-tight text-transparent bg-clip-text bg-gradient-to-b from-white to-white/60 sm:text-4xl font-display mb-3">
                暂无图片
              </h3>
              <p class="mt-2 max-w-sm mx-auto text-lg text-white/50 leading-relaxed font-light">
                此分享中暂时没有图片
              </p>
            </div>
          </div>

          <div v-else class="grid grid-cols-2 gap-1.5 sm:gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
            <div
                v-for="(image, index) in imageStore.images"
                :key="image?.id || index"
                :ref="(el) => setItemRef(el as HTMLElement | null, index)"
                :data-index="index"
                class="group relative aspect-square cursor-pointer overflow-hidden rounded-lg sm:rounded-2xl bg-white/5 transition-all duration-300 hover:scale-[1.02] hover:z-10"
                @click="handleImageClick(image, index)"
                @contextmenu="image && handleImageContextMenu($event, image, index)"
            >
              <div
                  class="absolute inset-0 rounded-lg sm:rounded-2xl ring-1 ring-inset ring-white/10 pointer-events-none transition-opacity group-hover:ring-white/20"
                  :class="[
                    uiStore.isSelectionMode && image && imageStore.selectedImages.has(image.id)
                      ? 'ring-2 ring-primary-500 bg-primary-500/10'
                      : ''
                  ]"></div>
              <template v-if="image">
                <img
                    :src="image.thumbnail_url || image.url"
                    :alt="image.original_name"
                    class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
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
                        ? 'border-primary-500 bg-primary-500 text-white'
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
                    @contextmenu.prevent="handleImageContextMenu($event, image, index)"
                >
                  <div class="absolute bottom-0 left-0 right-0 p-3">
                    <p class="text-white text-sm truncate">{{ image.original_name }}</p>
                  </div>
                </div>
              </template>
              <template v-else>
                <div class="h-full w-full animate-pulse flex items-center justify-center bg-white/5">
                  <PhotoIcon class="h-8 w-8 text-white/10"/>
                </div>
              </template>
            </div>
          </div>

        </div>
      </main>
    </div>

    <!-- 错误状态 -->
    <div v-else class="flex min-h-screen flex-col items-center justify-center p-4">
      <div class="text-center relative z-10">
        <div class="relative group mx-auto mb-8">
          <!-- 背景光晕 - 红色/橙色预警色调 -->
          <div
              class="absolute -inset-4 rounded-full bg-red-500/20 blur-2xl opacity-50 group-hover:opacity-75 transition-all duration-700"></div>

          <!-- 图标容器 -->
          <div
              class="relative mx-auto flex h-32 w-32 items-center justify-center rounded-full bg-white/5 ring-1 ring-white/10 backdrop-blur-xl shadow-2xl transition-all duration-500 group-hover:scale-105 group-hover:bg-white/10 group-hover:ring-white/20">
            <ExclamationTriangleIcon class="h-16 w-16 text-red-400/90 drop-shadow-[0_0_15px_rgba(248,113,113,0.5)]"/>
          </div>
        </div>

        <h3 class="text-3xl font-bold tracking-tight text-transparent bg-clip-text bg-gradient-to-b from-white to-white/60 sm:text-4xl font-display mb-3">
          无法访问
        </h3>
        <p class="mt-2 max-w-sm mx-auto text-lg text-white/50 leading-relaxed font-light">
          {{ error || '分享链接不存在或已过期' }}
        </p>
      </div>
    </div>

    <!-- 复用 ImageViewer 组件 -->
    <ImageViewer/>
  </div>
</template>

<script setup lang="ts">
import {nextTick, onMounted, onUnmounted, ref} from 'vue'
import {useRoute} from 'vue-router'
import {shareApi} from '@/api/share'
import {imageApi} from '@/api/image'
import {useImageStore} from '@/stores/image'
import {useUIStore} from '@/stores/ui'
import ImageViewer from '@/components/gallery/ImageViewer.vue'
import {
  ArchiveBoxArrowDownIcon,
  ArrowDownTrayIcon,
  CheckIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  EyeIcon,
  LockClosedIcon,
  PhotoIcon,
  SparklesIcon
} from '@heroicons/vue/24/outline'
import ContextMenu from '@/components/common/ContextMenu.vue'
import ContextMenuItem from '@/components/common/ContextMenuItem.vue'
import SelectionBox from '@/components/common/SelectionBox.vue'
import type {Image, SharePublicInfo} from '@/types'
import {useGenericBoxSelection} from '@/composables/useGenericBoxSelection'

const route = useRoute()
const imageStore = useImageStore()
const uiStore = useUIStore()
const code = route.params.code as string

const verifying = ref(false)
const needPassword = ref(false)
const shareInfo = ref<SharePublicInfo | null>(null)
const password = ref('')
const error = ref('')

// Context Menu State
const contextMenu = ref({visible: false, x: 0, y: 0})
const contextMenuTargetIds = ref<number[]>([])
const contextMenuSingleTarget = ref<{ image: Image, index: number } | null>(null)

const scrollContainerRef = ref<HTMLElement | null>(null)
const observer = ref<IntersectionObserver | null>(null)
const itemRefs = new Map<number, HTMLElement>()
const loadingPages = new Set<number>()

const {
  selectionBoxStyle,
  handleMouseDown,
  isDragOperation
} = useGenericBoxSelection<Image | null>({
  containerRef: scrollContainerRef,
  itemRefs,
  getItems: () => imageStore.images,
  getItemId: (item) => item?.id ?? -1,
  toggleSelection: (id) => {
    if (id === -1) return
    imageStore.toggleSelect(id)
  },
  onSelectionEnd: () => {
    uiStore.setSelectionMode(true)
  },
  useScroll: true
})

// 分享页面固定 6 列，计算合适的分页大小
// 6 列 * 5 行/屏 * 2 屏 = 60
const pageSize = 60

// 选择模式相关方法
function enterSelectionMode() {
  uiStore.setSelectionMode(true)
}

function exitSelectionMode() {
  uiStore.setSelectionMode(false)
  imageStore.clearSelection()
}

function handleImageClick(image: Image | null, index: number) {
  if (!image) return
  // 如果是拖拽操作结束，不处理点击
  if (isDragOperation()) return

  if (uiStore.isSelectionMode) {
    imageStore.toggleSelect(image.id)
  } else {
    imageStore.viewerIndex = index
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

// Context Menu Handlers
const handleImageContextMenu = (e: MouseEvent, image: Image, index: number) => {
  // Prevent browser context menu
  e.preventDefault()
  e.stopPropagation() // Important to prevent bubbling

  contextMenu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY
  }

  if (imageStore.selectedImages.has(image.id)) {
    contextMenuTargetIds.value = Array.from(imageStore.selectedImages)
  } else {
    // If right clicking an unselected image, select it properly
    contextMenuTargetIds.value = [image.id]
    // Optional: Clear selection and select this one if we want to behave like OS file explorers
    // For now, we just act on this one + context menu doesn't change selection state unless clicked
  }

  contextMenuSingleTarget.value = {image, index}
}

const handleContextMenuView = () => {
  if (contextMenuSingleTarget.value) {
    imageStore.viewerIndex = contextMenuSingleTarget.value.index
  }
  contextMenu.value.visible = false
}

const handleContextMenuDownload = () => {
  contextMenu.value.visible = false
  for (let targetId of contextMenuTargetIds.value) {
    if (targetId === undefined) continue;

    const img = imageStore.images.find(i => i?.id === targetId)
    if (img) imageApi.download(targetId, img.original_name)
  }
}

const handleContextMenuZipDownload = () => {
  contextMenu.value.visible = false
  imageApi.downloadZipped(contextMenuTargetIds.value.filter((id): id is number => id !== undefined))
}

async function downloadSelected() {
  if (imageStore.selectedCount === 0) return
  for (let targetId of imageStore.selectedImages) {
    if (!targetId) continue;

    const img = imageStore.images.find(i => i?.id === targetId)
    if (img) await imageApi.download(targetId, img.original_name)
  }
}

async function handleVerify() {
  verifying.value = true
  error.value = ''

  try {
    await initFetcher();
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
      await initFetcher()
    }
  } catch (err: any) {
    error.value = err.message || '获取分享信息失败'
  }
}

async function initFetcher() {
  // 设置 fetcher 并加载图片
  await imageStore.refreshImages(
      async (page, size) => (await shareApi.getImages(code, password.value, page, size)).data,
      pageSize
  )
  await nextTick()
  initObserver()
}

  // 先设置 fetcher，但不立即加载图片
  // 需要先检查分享信息，确认是否需要密码
onMounted(checkShare)

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

/*noinspection CssInvalidPropertyValue,CssInvalidFunction*/
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
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
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
