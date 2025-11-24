<template>
  <div class="p-3">
    <!-- 加载状态 -->
    <div v-if="imageStore.loading && (!imageStore.images || imageStore.images.length === 0)" class="py-12 text-center">
      <div class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-blue-600"></div>
      <p class="mt-4 text-sm text-gray-600">加载中...</p>
    </div>

    <!-- 空状态 -->
    <div v-else-if="!imageStore.images || imageStore.images.length === 0"
         class="flex min-h-[60vh] items-center justify-center py-20">
      <div class="text-center">
        <div class="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-gray-100">
          <PhotoIcon class="h-12 w-12 text-gray-400"/>
        </div>
        <h3 class="mt-6 text-xl font-semibold text-gray-900">还没有图片</h3>
        <p class="mt-3 text-base text-gray-600">上传您的第一张图片开始使用</p>
      </div>
    </div>

    <!-- 图片网格 -->
    <div ref="containerRef" v-else class="relative select-none" @mousedown="handleMouseDown">
      <!-- 右键菜单 -->
      <ContextMenu v-model="contextMenu.visible" :x="contextMenu.x" :y="contextMenu.y">
        <ContextMenuItem v-if="contextMenuTargetIds.length === 1" :icon="EyeIcon" @click="handleView">
          查看
        </ContextMenuItem>
        <ContextMenuItem :icon="ShareIcon" @click="handleShare">
          分享 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
        </ContextMenuItem>
        <ContextMenuItem :icon="PencilIcon" @click="handleEdit">
          编辑元数据 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
        </ContextMenuItem>
        <ContextMenuItem :icon="ArrowDownTrayIcon" @click="handleDownload">
          下载 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
        </ContextMenuItem>
        <ContextMenuItem :icon="TrashIcon" :danger="true" @click="handleDelete">
          删除 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
        </ContextMenuItem>
      </ContextMenu>

      <MetadataEditor
          v-model="isMetadataEditorOpen"
          :image-ids="metadataEditorTargetIds"
          :initial-data="metadataEditorInitialData"
          @saved="onMetadataSaved"
      />

      <CreateShare
          :is-open="isShareModalOpen"
          :selected-count="shareTargetIds.length"
          :selected-ids="shareTargetIds"
          @close="isShareModalOpen = false"
      />

      <!-- 框选区域 -->
      <div
          v-if="isSelecting"
          class="absolute z-50 border border-blue-500 bg-blue-500/20"
          :style="selectionBoxStyle"
      ></div>

      <!-- 瀑布流布局 -->
      <div v-if="isWaterfall" class="flex gap-4">
        <div
            v-for="(col, colIndex) in waterfallImages"
            :key="colIndex"
            class="flex-1 flex flex-col gap-4"
        >
          <div
              v-for="item in col"
              :key="item.index"
              class="relative group"
              :ref="(el) => setItemRef(el, item.index)"
              :data-index="item.index"
          >
            <template v-if="item.image">
              <ImageCard
                  :image="item.image"
                  :index="item.index"
                  :square="false"
                  @click="handleImageClick(item.image, item.index)"
                  @contextmenu="handleContextMenu($event, item.image, item.index)"
              />

              <!-- 选择模式遮罩 -->
              <div
                  v-if="uiStore.isSelectionMode"
                  class="absolute inset-0 cursor-pointer rounded-lg transition-colors"
                  :class="[
                  imageStore.selectedImages.has(item.image.id)
                    ? 'bg-blue-500/20 ring-2 ring-blue-500'
                    : 'hover:bg-black/10'
                ]"
                  @click.stop="handleImageClick(item.image, item.index)"
                  @contextmenu.prevent="handleContextMenu($event, item.image, item.index)"
              >
                <div class="absolute top-2 right-2">
                  <div
                      class="flex h-6 w-6 items-center justify-center rounded-full border-2 transition-colors"
                      :class="[
                      imageStore.selectedImages.has(item.image.id)
                        ? 'border-blue-500 bg-blue-500 text-white'
                        : 'border-white/70 bg-black/20 hover:bg-black/40'
                    ]"
                  >
                    <CheckIcon v-if="imageStore.selectedImages.has(item.image.id)" class="h-4 w-4"/>
                  </div>
                </div>
              </div>
            </template>

            <!-- 占位符 -->
            <div v-else
                 class="w-full animate-pulse rounded-lg bg-gray-200 flex items-center justify-center min-h-[200px]">
              <PhotoIcon class="h-8 w-8 text-gray-300"/>
            </div>
          </div>
        </div>
      </div>

      <!-- 网格布局 -->
      <div
          v-else
          :class="[
          'grid gap-2',
          gridClass,
        ]"
      >
        <div
            v-for="(image, index) in imageStore.images"
            :key="index"
            class="relative group aspect-square"
            :ref="(el) => setItemRef(el, index)"
            :data-index="index"
        >
          <template v-if="image">
            <ImageCard
                :image="image"
                :index="index"
                @click="handleImageClick(image, index)"
                @contextmenu="handleContextMenu($event, image, index)"
            />

            <!-- 选择模式遮罩 -->
            <div
                v-if="uiStore.isSelectionMode"
                class="absolute inset-0 cursor-pointer rounded-lg transition-colors"
                :class="[
                imageStore.selectedImages.has(image.id)
                  ? 'bg-blue-500/20 ring-2 ring-blue-500'
                  : 'hover:bg-black/10'
              ]"
                @click.stop="handleImageClick(image, index)"
                @contextmenu.prevent="handleContextMenu($event, image, index)"
            >
              <div class="absolute top-2 right-2">
                <div
                    class="flex h-6 w-6 items-center justify-center rounded-full border-2 transition-colors"
                    :class="[
                    imageStore.selectedImages.has(image.id)
                      ? 'border-blue-500 bg-blue-500 text-white'
                      : 'border-white/70 bg-black/20 hover:bg-black/40'
                  ]"
                >
                  <CheckIcon v-if="imageStore.selectedImages.has(image.id)" class="h-4 w-4"/>
                </div>
              </div>
            </div>
          </template>

          <!-- 占位符 -->
          <div v-else class="h-full w-full animate-pulse rounded-lg bg-gray-200 flex items-center justify-center">
            <PhotoIcon class="h-8 w-8 text-gray-300"/>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- 图片查看器 -->
  <ImageViewer/>
</template>

<script setup lang="ts">
import type {ComponentPublicInstance} from 'vue'
import {computed, onMounted, onUnmounted, ref, watch} from 'vue'
import {useDebounceFn, useThrottleFn} from '@vueuse/core'
import {useImageStore} from '@/stores/image'
import {useUIStore} from '@/stores/ui'
import {imageApi} from '@/api/image'
import type {Image} from '@/types'
import {
  ArrowDownTrayIcon,
  CheckIcon,
  EyeIcon,
  PencilIcon,
  PhotoIcon,
  TrashIcon,
  ShareIcon
} from '@heroicons/vue/24/outline'
import ImageCard from './ImageCard.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import ContextMenuItem from '@/components/common/ContextMenuItem.vue'
import MetadataEditor from './menu/MetadataEditor.vue'
import CreateShare from '@/components/gallery/menu/CreateShare.vue'
import ImageViewer from "@/components/gallery/ImageViewer.vue";

const index = ref<number>(-1)

const imageStore = useImageStore()
const uiStore = useUIStore()
const containerRef = ref<HTMLElement>()

// 右键菜单状态
const contextMenu = ref({visible: false, x: 0, y: 0})
const contextMenuTargetIds = ref<number[]>([])
const contextMenuSingleTarget = ref<{ image: Image, index: number } | null>(null)

// 元数据编辑器状态
const isMetadataEditorOpen = ref(false)
const metadataEditorTargetIds = ref<number[]>([])
const metadataEditorInitialData = ref<Image | null>(null)

// 分享弹窗状态
const isShareModalOpen = ref(false)
const shareTargetIds = ref<number[]>([])

const handleContextMenu = (e: MouseEvent, image: Image, index: number) => {
  contextMenu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY
  }

  if (imageStore.selectedImages.has(image.id)) {
    contextMenuTargetIds.value = Array.from(imageStore.selectedImages)
  } else {
    contextMenuTargetIds.value = [image.id]
  }

  contextMenuSingleTarget.value = {image, index}
}

const handleShare = () => {
  shareTargetIds.value = contextMenuTargetIds.value
  isShareModalOpen.value = true
  contextMenu.value.visible = false
}

const handleView = () => {
  if (contextMenuSingleTarget.value) {
    index.value = contextMenuSingleTarget.value.index
  }
  contextMenu.value.visible = false
}

const handleEdit = () => {
  metadataEditorTargetIds.value = contextMenuTargetIds.value
  if (contextMenuTargetIds.value.length === 1 && contextMenuSingleTarget.value && contextMenuTargetIds.value[0] === contextMenuSingleTarget.value.image.id) {
    metadataEditorInitialData.value = contextMenuSingleTarget.value.image
  } else if (contextMenuTargetIds.value.length === 1) {
    // Fallback if logic matches
    const img = imageStore.images.find(i => i?.id === contextMenuTargetIds.value[0])
    metadataEditorInitialData.value = img || null
  } else {
    metadataEditorInitialData.value = null
  }
  isMetadataEditorOpen.value = true
  contextMenu.value.visible = false
}

const handleDownload = async () => {
  contextMenu.value.visible = false
  // Loop download
  for (const id of contextMenuTargetIds.value) {
    const img = imageStore.images.find(i => i?.id === id)
    if (img) {
      await imageApi.download(id, img.original_name)
    }
  }
}

const handleDelete = async () => {
  contextMenu.value.visible = false
  if (!confirm(`Are you sure you want to delete ${contextMenuTargetIds.value.length} images?`)) return

  try {
    await imageApi.deleteBatch(contextMenuTargetIds.value)
    // Refresh or remove from store
    // Assuming store has a remove method or we just fetch again
    // imageStore.removeImages(contextMenuTargetIds.value) // If exists
    // Or fetch
    await imageStore.fetchImages(1) // Simple reload for now
    imageStore.selectedImages.clear()
  } catch (e) {
    console.error('Delete failed', e)
  }
}

const onMetadataSaved = () => {
  // Refresh list to show new data
  // Ideally update local data, but fetching is safer
  imageStore.fetchImages(imageStore.currentPage) // Stay on current page
}

// 瀑布流相关
const isWaterfall = computed(() => uiStore.gridDensity >= 8)
const currentColumnCount = ref(4)

const waterfallImages = computed(() => {
  if (!isWaterfall.value) return []

  const cols: { image: Image | null, index: number }[][] =
      Array.from({length: currentColumnCount.value}, () => [])

  imageStore.images.forEach((image, index) => {
    const colIndex = index % currentColumnCount.value
    if (cols[colIndex]) cols[colIndex].push({image, index})
  })

  return cols
})

function updateColumnCount() {
  const width = window.innerWidth
  const cols = uiStore.gridColumns
  if (width >= 768) {
    currentColumnCount.value = cols.desktop
  } else if (width >= 640) {
    currentColumnCount.value = cols.tablet
  } else {
    currentColumnCount.value = cols.mobile
  }
}

// 监听 gridColumns 变化，更新列数
watch(() => uiStore.gridColumns, () => {
  updateColumnCount()
})

// 框选相关状态
const isSelecting = ref(false)
const selectionStart = ref({x: 0, y: 0})
const selectionCurrent = ref({x: 0, y: 0})
const lastMousePos = ref({x: 0, y: 0})
const itemRefs = new Map<number, HTMLElement>() // Key is index
const itemRects = new Map<number, { left: number, top: number, right: number, bottom: number }>() // Key is index
const initialSelection = new Set<number>()
let isDragOperation = false
const observer = ref<IntersectionObserver | null>(null)
const scrollContainer = ref<HTMLElement | null>(null)

const selectionBoxStyle = computed(() => {
  if (!isSelecting.value) return {}

  const left = Math.min(selectionStart.value.x, selectionCurrent.value.x)
  const top = Math.min(selectionStart.value.y, selectionCurrent.value.y)
  const width = Math.abs(selectionCurrent.value.x - selectionStart.value.x)
  const height = Math.abs(selectionCurrent.value.y - selectionStart.value.y)

  return {
    left: `${left}px`,
    top: `${top}px`,
    width: `${width}px`,
    height: `${height}px`
  }
})

function setItemRef(el: Element | ComponentPublicInstance | null, index: number) {
  if (el) {
    const element = el as HTMLElement
    itemRefs.set(index, element)
    // 如果 observer 已创建，立即观察
    if (observer.value) {
      observer.value.observe(element)
    }
  } else {
    itemRefs.delete(index)
  }
}

function handleMouseDown(e: MouseEvent) {
  // 仅在选择模式下，且使用左键点击时触发
  if (e.button !== 0 || !containerRef.value) return
  // 阻止默认事件，防止选中文本
  // 但不阻止冒泡，否则可能影响其他交互？
  // e.preventDefault()

  isDragOperation = false

  const containerRect = containerRef.value.getBoundingClientRect()
  const startX = e.clientX - containerRect.left
  const startY = e.clientY - containerRect.top

  selectionStart.value = {x: startX, y: startY}
  selectionCurrent.value = {x: startX, y: startY}
  lastMousePos.value = {x: e.clientX, y: e.clientY}

  // 缓存所有图片位置 (相对于容器)
  itemRects.clear()
  itemRefs.forEach((el, index) => {
    itemRects.set(index, {
      left: el.offsetLeft,
      top: el.offsetTop,
      right: el.offsetLeft + el.offsetWidth,
      bottom: el.offsetTop + el.offsetHeight
    })
  })

  // 记录初始选中状态
  initialSelection.clear()
  imageStore.selectedImages.forEach(id => initialSelection.add(id))

  window.addEventListener('mousemove', handleMouseMove)
  window.addEventListener('mouseup', handleMouseUp)
  window.addEventListener('scroll', handleScroll, true)
}

function updateSelectionPos(clientX: number, clientY: number) {
  if (!containerRef.value) return

  const containerRect = containerRef.value.getBoundingClientRect()
  selectionCurrent.value = {
    x: clientX - containerRect.left,
    y: clientY - containerRect.top
  }

  updateSelection()
}

function handleMouseMove(e: MouseEvent) {
  lastMousePos.value = {x: e.clientX, y: e.clientY}

  if (!isSelecting.value) {
    if (!containerRef.value) return
    const containerRect = containerRef.value.getBoundingClientRect()
    const startX = selectionStart.value.x + containerRect.left
    const startY = selectionStart.value.y + containerRect.top

    // 判断是否达到拖拽阈值 (5px)
    const dx = e.clientX - startX
    const dy = e.clientY - startY
    if (dx * dx + dy * dy > 25) {
      isSelecting.value = true
    } else {
      return
    }
  }

  updateSelectionPos(e.clientX, e.clientY)
}

function handleMouseUp() {
  window.removeEventListener('mousemove', handleMouseMove)
  window.removeEventListener('mouseup', handleMouseUp)
  window.removeEventListener('scroll', handleScroll, true)

  if (isSelecting.value) {
    uiStore.setSelectionMode(true)
    isSelecting.value = false
    isDragOperation = true
    // 延迟重置拖拽标志，确保在 click 事件触发时 flag 为 true
    setTimeout(() => {
      isDragOperation = false
    }, 0)
  }
}

function updateSelection() {
  const left = Math.min(selectionStart.value.x, selectionCurrent.value.x)
  const top = Math.min(selectionStart.value.y, selectionCurrent.value.y)
  const right = Math.max(selectionStart.value.x, selectionCurrent.value.x)
  const bottom = Math.max(selectionStart.value.y, selectionCurrent.value.y)

  // 基于初始选中状态计算
  const newSelection = new Set(initialSelection)

  itemRects.forEach((rect, index) => {
    // 判断矩形相交
    const isIntersecting = !(
        rect.right < left ||
        rect.left > right ||
        rect.bottom < top ||
        rect.top > bottom
    )

    if (isIntersecting) {
      const image = imageStore.images[index]
      if (image) {
        // 如果原本已选中，则取消选中；如果原本未选中，则选中
        if (initialSelection.has(image.id)) {
          newSelection.delete(image.id)
        } else {
          newSelection.add(image.id)
        }
      }
    }
  })

  // 更新 store
  imageStore.selectedImages = newSelection
}

// 根据密度动态计算网格列数
const gridClass = computed(() => {
  const columns = uiStore.gridColumns
  const desktopClass = {
    1: 'md:grid-cols-1',
    2: 'md:grid-cols-2',
    3: 'md:grid-cols-3',
    4: 'md:grid-cols-4',
    5: 'md:grid-cols-5',
    6: 'md:grid-cols-6',
    7: 'md:grid-cols-7',
    8: 'md:grid-cols-8',
    9: 'md:grid-cols-9',
  }[columns.desktop] || 'md:grid-cols-4'

  const tabletClass = {
    1: 'sm:grid-cols-1',
    2: 'sm:grid-cols-2',
    3: 'sm:grid-cols-3',
    4: 'sm:grid-cols-4',
  }[columns.tablet] || 'sm:grid-cols-2'

  const mobileClass = columns.mobile === 1 ? 'grid-cols-1' : 'grid-cols-2'

  return `${mobileClass} ${tabletClass} ${desktopClass}`
})

function handleImageClick(image: Image, index: number) {
  // 如果是拖拽操作结束，不处理点击
  if (isDragOperation) return

  if (uiStore.isSelectionMode) {
    imageStore.toggleSelect(image.id)
  } else {
    imageStore.viewerIndex = index
  }
}


// 更新时间线状态
const updateActiveDate = useThrottleFn(() => {
  if (!scrollContainer.value) return

  const container = scrollContainer.value
  const rect = container.getBoundingClientRect()

  // 检测视口顶部偏下一点的位置 (比如 100px)
  const checkX = rect.left + 100 // 稍微往右一点，避开可能的边缘
  const checkY = rect.top + 100

  const el = document.elementFromPoint(checkX, checkY)
  if (!el) return

  // 向上查找包含 data-index 的元素
  const itemEl = el.closest('[data-index]') as HTMLElement
  if (itemEl && itemEl.dataset.index) {
    const image = imageStore.images[parseInt(itemEl.dataset.index)] as (Image | null)
    if (image) {
      // 优先使用拍摄时间，否则使用创建时间
      const date = image.taken_at || image.created_at
      if (date && date !== uiStore.timeLineState?.date) {
        uiStore.setTimeLineState({date, location: image.location_name})
      }
    }
  }
}, 100)

// 停止滚动 1.5 秒后隐藏时间线
const hideTimeline = useDebounceFn(() => {
  uiStore.setTimeLineState(null)
}, 1500)

// 滚动处理函数
const handleScroll = () => {
  updateActiveDate()
  hideTimeline()
}

onMounted(() => {
  updateColumnCount()
  window.addEventListener('resize', updateColumnCount)

  // 获取滚动容器并添加监听
  scrollContainer.value = document.getElementById('main-scroll-container')
  if (scrollContainer.value) {
    scrollContainer.value.addEventListener('scroll', handleScroll)
    // 初始化一次，并安排自动隐藏
    setTimeout(() => handleScroll(), 100)
  }

  // 初始化 IntersectionObserver
  observer.value = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
          if (entry.isIntersecting) {
            const index = Number((entry.target as HTMLElement).dataset.index)
            if (!isNaN(index)) {
              // 如果该位置没有图片，触发加载
              if (!imageStore.images[index]) {
                const pageSize = uiStore.pageSize
                const page = Math.floor(index / pageSize) + 1
                imageStore.fetchImages(page, pageSize)
              }
            }
          }
        })
      },
      {
        rootMargin: '200px 0px', // 提前 200px 加载
        threshold: 0
      }
  )

  // 观察所有已存在的元素
  itemRefs.forEach((el) => {
    observer.value?.observe(el)
  })
})

onUnmounted(() => {
  window.removeEventListener('resize', updateColumnCount)

  if (scrollContainer.value) {
    scrollContainer.value.removeEventListener('scroll', handleScroll)
  }

  observer.value?.disconnect()
  observer.value = null
})
</script>
