<template>
  <div class="p-3">
    <!-- 加载状态 -->
    <div v-if="imageStore.loading && (!imageStore.images || imageStore.images.length === 0)"
         class="flex min-h-[60vh] items-center justify-center">
      <div class="text-center">
        <div
            class="inline-block h-12 w-12 animate-spin rounded-full border-2 border-white/5 border-t-primary-500 shadow-[0_0_15px_rgba(139,92,246,0.3)]"></div>
        <p class="mt-6 text-sm text-white/40 tracking-[0.2em] uppercase font-medium">Loading</p>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else-if="!imageStore.images || imageStore.images.length === 0"
         class="flex min-h-[60vh] flex-col items-center justify-center py-20">
      <div class="text-center relative z-10" style="user-select: none">
        <div class="relative group mx-auto mb-8">
          <!-- 背景光晕 -->
          <div
              class="absolute -inset-4 rounded-full bg-primary-500/20 blur-2xl opacity-50 group-hover:opacity-75 transition-all duration-700"></div>

          <!-- 图标容器 -->
          <div
              class="relative mx-auto flex h-32 w-32 items-center justify-center rounded-full bg-white/5 ring-1 ring-white/10 backdrop-blur-xl shadow-2xl transition-all duration-500 group-hover:scale-105 group-hover:bg-white/10 group-hover:ring-white/20">
            <SparklesIcon class="h-16 w-16 text-primary-400/90 drop-shadow-[0_0_15px_rgba(139,92,246,0.5)]"/>
          </div>

          <!-- 装饰元素 -->
          <div
              class="absolute top-0 right-0 h-8 w-8 rounded-full bg-white/5 ring-1 ring-white/10 backdrop-blur-md flex items-center justify-center animate-bounce"
              style="animation-duration: 3s;">
            <CloudArrowUpIcon class="h-4 w-4 text-primary-300"/>
          </div>
        </div>

        <h3 class="text-3xl font-bold tracking-tight text-transparent bg-clip-text bg-gradient-to-b from-white to-white/60 sm:text-4xl font-display mb-3">
          暂无图片
        </h3>
        <p class="mt-2 max-w-sm mx-auto text-lg text-white/50 leading-relaxed font-light">
          上传您的第一张图片，开启您的光影画廊
        </p>
      </div>
    </div>

    <!-- 图片网格 -->
    <div ref="containerRef" v-else class="relative select-none" @mousedown="handleMouseDown">
      <!-- 右键菜单 -->
      <ContextMenu v-model="contextMenu.visible" :x="contextMenu.x" :y="contextMenu.y">
        <template v-if="props.mode === 'gallery'">
          <template v-if="props.excludeAlbumId">
            <!-- 相册模式特有菜单 -->
            <ContextMenuItem v-if="contextMenuTargetIds.length === 1" :icon="PhotoIcon" @click="handleSetAlbumCover">
              设为封面
            </ContextMenuItem>
            <ContextMenuItem :icon="MinusCircleIcon" :danger="true" @click="handleRemoveFromAlbum">
              从相册移除 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
            </ContextMenuItem>
            <div class="h-px bg-white/10 my-1"></div>
          </template>

          <ContextMenuItem :icon="RectangleStackIcon" @click="handleAddToAlbum">
            添加到相册 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
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
          <ContextMenuItem v-if="contextMenuTargetIds.length>1" :icon="ArchiveBoxArrowDownIcon" @click="handleBatchDownload">
            打包下载 {{  `(${contextMenuTargetIds.length})` }}
          </ContextMenuItem>
          <ContextMenuItem :icon="TrashIcon" :danger="true" @click="handleDelete">
            删除 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
          </ContextMenuItem>
        </template>

        <template v-else-if="props.mode === 'trash'">
          <ContextMenuItem :icon="ArrowUturnLeftIcon" @click="handleRestore">
            恢复 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
          </ContextMenuItem>
          <ContextMenuItem :icon="XMarkIcon" :danger="true" @click="handlePermanentDelete">
            彻底删除 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
          </ContextMenuItem>
        </template>
      </ContextMenu>

      <MetadataEditor
          v-model="isMetadataEditorOpen"
          :image-ids="metadataEditorTargetIds"
          :initial-data="metadataEditorInitialData"
          @saved="onMetadataSaved"
      />

      <CreateShare
          v-model="isShareModalOpen"
          :selected-count="shareTargetIds.length"
          :selected-ids="shareTargetIds"
      />

      <AddToAlbumModal
          v-model="isAddToAlbumOpen"
          :image-ids="addToAlbumTargetIds"
          :exclude-album-id="props.excludeAlbumId"
          @added="onAddedToAlbum"
      />

      <!-- 框选区域 -->
      <SelectionBox :style="selectionBoxStyle"/>

      <!-- 瀑布流布局 -->
      <div v-show="isWaterfall" class="flex gap-4">
        <div
            v-for="(col, colIndex) in waterfallImages"
            :key="colIndex"
            class="flex-1 flex flex-col gap-4"
        >
          <div
              v-for="item in col"
              :key="item.image?.id ?? `placeholder-${item.index}`"
              class="relative group"
              :ref="(el) => isWaterfall && setItemRef(el, item.index)"
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
                    ? 'bg-primary-500/20 ring-2 ring-primary-500'
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
                        ? 'border-primary-500 bg-primary-500 text-white'
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
                 class="w-full animate-pulse rounded-2xl bg-white/5 ring-1 ring-white/10 flex items-center justify-center min-h-[200px]">
              <PhotoIcon class="h-8 w-8 text-white/10"/>
            </div>
          </div>
        </div>
      </div>

      <!-- 网格布局 -->
      <div
          v-show="!isWaterfall"
          :class="[
          'grid gap-2',
          gridClass,
        ]"
      >
        <div
            v-for="(image, index) in imageStore.images"
            :key="image?.id ?? `placeholder-${index}`"
            class="relative group aspect-square"
            :ref="(el) => !isWaterfall && setItemRef(el, index)"
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
                class="absolute inset-0 cursor-pointer rounded-2xl transition-colors"
                :class="[
                imageStore.selectedImages.has(image.id)
                  ? 'bg-primary-500/20 ring-2 ring-primary-500'
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
                      ? 'border-primary-500 bg-primary-500 text-white'
                      : 'border-white/70 bg-black/20 hover:bg-black/40'
                  ]"
                >
                  <CheckIcon v-if="imageStore.selectedImages.has(image.id)" class="h-4 w-4"/>
                </div>
              </div>
            </div>
          </template>

          <!-- 占位符 -->
          <div v-else
               class="h-full w-full animate-pulse rounded-2xl bg-white/5 ring-1 ring-white/10 flex items-center justify-center">
            <PhotoIcon class="h-8 w-8 text-white/10"/>
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
import {useDialogStore} from '@/stores/dialog'
import {imageApi} from '@/api/image'
import type {Image} from '@/types'
import {
  ArchiveBoxArrowDownIcon,
  ArrowDownTrayIcon,
  ArrowUturnLeftIcon,
  CheckIcon,
  CloudArrowUpIcon,
  MinusCircleIcon,
  PencilIcon,
  PhotoIcon,
  RectangleStackIcon,
  ShareIcon,
  SparklesIcon,
  TrashIcon,
  XMarkIcon
} from '@heroicons/vue/24/outline'
import ImageCard from './ImageCard.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import ContextMenuItem from '@/components/common/ContextMenuItem.vue'
import SelectionBox from '@/components/common/SelectionBox.vue'
import MetadataEditor from './menu/MetadataEditor.vue'
import CreateShare from '@/components/gallery/menu/CreateShare.vue'
import AddToAlbumModal from '@/components/album/AddToAlbumModal.vue'
import ImageViewer from "@/components/gallery/ImageViewer.vue";
import {useGenericBoxSelection} from '@/composables/useGenericBoxSelection'
import {useAlbumStore} from '@/stores/album'
import {albumApi} from '@/api/album'

const props = withDefaults(defineProps<{
  mode?: 'gallery' | 'trash'
  excludeAlbumId?: number  // 添加到相册时排除的相册ID
}>(), {
  mode: 'gallery'
})


const imageStore = useImageStore()
const albumStore = useAlbumStore()
const uiStore = useUIStore()
const dialogStore = useDialogStore()
const containerRef = ref<HTMLElement>()

// ... (previous refs)

// ImageGrid actions for album mode
async function handleSetAlbumCover() {
  contextMenu.value.visible = false
  if (!props.excludeAlbumId || contextMenuTargetIds.value.length !== 1) return

  try {
    await albumStore.setAlbumCover(props.excludeAlbumId, contextMenuTargetIds.value[0] as number)
    // Optional: show toast/notification success
    dialogStore.alert({title: '成功', message: '设置封面成功',type:'success'})
  } catch (err) {
    console.error('设置封面失败', err)
    dialogStore.alert({title: '错误', message: '设置封面失败', type: 'error'})
  }
}

async function handleRemoveFromAlbum() {
  contextMenu.value.visible = false
  if (!props.excludeAlbumId || contextMenuTargetIds.value.length === 0) return

  const ids = contextMenuTargetIds.value
  const albumId = props.excludeAlbumId

  try {
    await albumApi.removeImages(albumId, ids)
    // 更新本地列表
    imageStore.images = imageStore.images.filter(img => img === null || !ids.includes(img.id))
    imageStore.total -= ids.length
    imageStore.selectedImages.clear() // Clear selection if any were removed (though typically context menu targets selection)

    // 更新当前相册状态（如果在 AlbumDetailView 中）
    if (albumStore.currentAlbum && albumStore.currentAlbum.id === albumId) {
      albumStore.currentAlbum.image_count -= ids.length
    }
  } catch (err) {
    console.error('从相册移除失败', err)
    dialogStore.alert({title: '错误', message: '移除失败', type: 'error'})
  }
}

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

// 添加到相册弹窗状态
const isAddToAlbumOpen = ref(false)
const addToAlbumTargetIds = ref<number[]>([])

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

const handleAddToAlbum = () => {
  addToAlbumTargetIds.value = contextMenuTargetIds.value
  isAddToAlbumOpen.value = true
  contextMenu.value.visible = false
}

const onAddedToAlbum = () => {
  // 添加成功后清除选择
  imageStore.selectedImages.clear()
  uiStore.setSelectionMode(false)
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

const handleDownload = () => {
  contextMenu.value.visible = false
  for (let targetId of contextMenuTargetIds.value) {
    if (targetId === undefined) continue;

    const img = imageStore.images.find(i => i?.id === targetId)
    if (img) imageApi.download(targetId, img.original_name)
  }
}

const handleBatchDownload = () => {
  contextMenu.value.visible = false
  // 多个文件使用批量下载
  imageApi.downloadZipped(contextMenuTargetIds.value.filter((id): id is number => id !== undefined))
}

const handleDelete = async () => {
  contextMenu.value.visible = false
  const confirmed = await dialogStore.confirm({
    title: 'Delete Images',
    message: `Are you sure you want to delete ${contextMenuTargetIds.value.length} images?`,
    type: 'warning',
    confirmText: 'Delete'
  })
  if (!confirmed) return

  try {
    await imageStore.deleteBatch()
    // Refresh or remove from store
    // Assuming store has a remove method or we just fetch again
    // imageStore.removeImages(contextMenuTargetIds.value) // If exists
    // Or fetch
    // await imageStore.fetchImages(1) // Simple reload for now
    imageStore.selectedImages.clear()
  } catch (e) {
    console.error('Delete failed', e)
  }
}

const handleRestore = async () => {
  contextMenu.value.visible = false
  try {
    await imageApi.restoreImages(contextMenuTargetIds.value)
    // 从列表移除已恢复的图片
    imageStore.images = imageStore.images.filter(img => img === null || !contextMenuTargetIds.value.includes(img.id))
    imageStore.total -= contextMenuTargetIds.value.length
    imageStore.selectedImages.clear()
  } catch (err) {
    console.error('恢复图片失败', err)
  }
}

const handlePermanentDelete = async () => {
  contextMenu.value.visible = false
  const confirmed = await dialogStore.confirm({
    title: '确认彻底删除',
    message: `确定要彻底删除 ${contextMenuTargetIds.value.length} 张图片吗？此操作不可恢复。`,
    type: 'error',
    confirmText: '彻底删除'
  })
  if (!confirmed) return

  try {
    await imageApi.permanentlyDelete(contextMenuTargetIds.value)
    // 从列表移除已删除的图片
    imageStore.images = imageStore.images.filter(img => img === null || !contextMenuTargetIds.value.includes(img.id))
    imageStore.total -= contextMenuTargetIds.value.length
    imageStore.selectedImages.clear()
  } catch (err) {
    console.error('彻底删除图片失败', err)
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

const itemRefs = new Map<number, HTMLElement>() // Key is index
const observer = ref<IntersectionObserver | null>(null)
const scrollContainer = ref<HTMLElement | null>(null)

const {
  selectionBoxStyle,
  handleMouseDown,
  isDragOperation
} = useGenericBoxSelection<Image | null>({
  containerRef,
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
  useScroll: false
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
  if (isDragOperation()) return

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
                const pageSize = uiStore.imagePageSize
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
