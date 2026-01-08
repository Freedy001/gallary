<template>
  <div class="p-3">
    <!-- 加载状态 -->
    <div v-if="imageList.loading.value && imageList.images.value.length === 0"
         class="flex min-h-[60vh] items-center justify-center">
      <div class="text-center">
        <div
            class="inline-block h-12 w-12 animate-spin rounded-full border-2 border-white/5 border-t-primary-500 shadow-glow"></div>
        <p class="mt-6 text-sm text-white/40 tracking-[0.2em] uppercase font-medium">Loading</p>
      </div>
    </div>

    <!-- 空状态 -->
    <ImageGridEmpty v-else-if="imageList.images.value.length === 0"/>

    <!-- 图片网格 -->
    <div ref="containerRef" v-else class="relative select-none" @mousedown="handleMouseDown">
      <!-- 右键菜单 -->
      <ImageGridContextMenu
          ref="contextMenuRef"
          :album-id="props.albumId"
          :images="imageList.images"
          :mode="props.mode"
          :on-refresh="() => imageList.fetchImages(imageList.currentPage.value)"
          :on-remove-images="imageList.removeImages"
          :selected-images="imageList.selectedImages"
      />

      <!-- 框选区域 -->
      <SelectionBox :style="selectionBoxStyle"/>

      <!-- 瀑布流布局 -->
      <div v-show="layout.isWaterfall.value" class="flex gap-4">
        <div
            v-for="(col, colIndex) in layout.waterfallImages.value"
            :key="colIndex"
            class="flex-1 flex flex-col gap-4"
        >
          <ImageGridItem
              v-for="item in col"
              :key="item.image?.id ?? `placeholder-${item.index}`"
              :ref="(el: any) => layout.isWaterfall.value && setItemRef(el, item.index)"
              :data-index="item.index"
              :image="item.image"
              :is-selected="item.image ? imageList.selectedImages.value.has(item.image.id) : false"
              :is-selection-mode="uiStore.isSelectionMode"
              :square="false"
              @click="handleImageClick(item.image,item.index)"
              @contextmenu.prevent="handleContextMenu($event,item.image,item.index)"
          />
        </div>
      </div>

      <!-- 网格布局 -->
      <div v-show="!layout.isWaterfall.value" :class="['grid gap-2', layout.gridClass.value]">
        <ImageGridItem
            v-for="(image, index) in imageList.images.value"
            :key="image?.id ?? `placeholder-${index}`"
            :ref="(el: any) => !layout.isWaterfall.value && setItemRef(el, index)"
            :data-index="index"
            :image="image"
            :is-selected="image ? imageList.selectedImages.value.has(image.id) : false"
            :is-selection-mode="uiStore.isSelectionMode"
            :square="true"
            @click="handleImageClick(image,index)"
            @contextmenu.prevent="handleContextMenu($event,image,index)"
        />
      </div>
    </div>
  </div>

  <!-- 图片查看器 -->
  <ImageViewer
      v-model:index="imageList.viewerIndex.value"
      :images="imageList.images.value"
      @delete="handleViewerDelete"
  />
</template>

<script setup lang="ts">
import type {ComponentPublicInstance} from 'vue'
import {onMounted, onUnmounted, ref, watch} from 'vue'
import {useUIStore} from '@/stores/ui'
import type {Image} from '@/types'
import SelectionBox from '@/components/widgets/common/SelectionBox.vue'
import ImageViewer from '@/components/gallery/ImageViewer.vue'
import ImageGridEmpty from '@/components/gallery/ImageGridEmpty.vue'
import ImageGridItem from '@/components/gallery/ImageGridItem.vue'
import ImageGridContextMenu from '@/components/gallery/ImageGridContextMenu.vue'
import {type Fether, useImageList} from '@/composables/useImageList'
import {useImageGridLayout} from '@/composables/useImageGridLayout'
import {useTimelineScroll} from '@/composables/useTimelineScroll'
import {useGenericBoxSelection} from '@/composables/useGenericBoxSelection'

// Props
const props = withDefaults(defineProps<{
  mode?: 'gallery' | 'trash'
  albumId?: number
  fetcher: Fether
}>(), {
  mode: 'gallery'
})

// Emits
const emit = defineEmits<{
  (e: 'update:total', total: number): void
  (e: 'update:selectedCount', count: number): void
  (e: 'update:loading', loading: boolean): void
}>()

// Stores
const uiStore = useUIStore()

// ==================== Composables ====================
const imageList = useImageList({
  originFetcher: props.fetcher,
  pageSize: uiStore.imagePageSize
})

const layout = useImageGridLayout({
  images: imageList.images
})

useTimelineScroll({
  images: imageList.images
})

// 右键菜单组件引用
const contextMenuRef = ref<InstanceType<typeof ImageGridContextMenu> | null>(null)

function handleContextMenu(e: MouseEvent, image: Image | null, index: number) {
  if (image) contextMenuRef.value?.handleContextMenu(e, image, index)
}

// 同步状态到父组件
watch(imageList.total, (val) => emit('update:total', val))
watch(imageList.selectedCount, (val) => emit('update:selectedCount', val))
watch(imageList.loading, (val) => emit('update:loading', val))

// ==================== 框选 ====================
const containerRef = ref<HTMLElement>()
const itemRefs = new Map<number, HTMLElement>()
const observer = ref<IntersectionObserver | null>(null)

const {
  selectionBoxStyle,
  handleMouseDown,
  isDragOperation
} = useGenericBoxSelection<Image | null>({
  containerRef,
  itemRefs,
  getItems: () => imageList.images.value,
  getItemId: (item) => item?.id ?? -1,
  toggleSelection: (id) => {
    if (id === -1) return
    imageList.toggleSelect(id)
  },
  onSelectionEnd: () => {
    uiStore.setSelectionMode(true)
  },
  useScroll: false
})

function setItemRef(el: Element | ComponentPublicInstance | null, index: number) {
  if (el) {
    const element = (el as any)?.$el ?? el as HTMLElement
    itemRefs.set(index, element)
    if (observer.value) {
      observer.value.observe(element)
    }
  } else {
    itemRefs.delete(index)
  }
}

function handleImageClick(image: Image | null, index: number) {
  if (isDragOperation()) return

  // 如果右键菜单正在显示，点击其他地方只关闭菜单，不触发图片查看
  if (contextMenuRef.value?.isVisible()) return

  if (uiStore.isSelectionMode) {
    if (image) imageList.toggleSelect(image.id)
  } else {
    imageList.viewerIndex.value = index
  }
}

function handleViewerDelete(id: number) {
  imageList.removeImages([id])

  if (imageList.viewerIndex.value >= imageList.images.value.length) {
    imageList.viewerIndex.value = imageList.images.value.length - 1
  }
  if (imageList.images.value.length === 0) {
    imageList.viewerIndex.value = -1
  }
}

// 暴露方法给父组件
defineExpose({
  refresh: imageList.refresh,
  selectAll: imageList.selectAll,
  clearSelection: imageList.clearSelection,
  deleteBatch: async () => {
    const idsToDelete = Array.from(imageList.selectedImages.value)
    if (idsToDelete.length === 0) return
    const {imageApi} = await import('@/api/image')
    await imageApi.deleteBatch(idsToDelete)
    imageList.removeImages(idsToDelete)
  },
  images: imageList.images,
  total: imageList.total,
  selectedCount: imageList.selectedCount,
  selectedIds: imageList.selectedIds,
})

// ==================== 生命周期 ====================
onMounted(async () => {
  observer.value = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        const index = Number((entry.target as HTMLElement).dataset.index)
        if (!isNaN(index) && !imageList.images.value[index]) {
          const pageSize = uiStore.imagePageSize
          const page = Math.floor(index / pageSize) + 1
          imageList.fetchImages(page, pageSize)
        }
      }
    })
  }, {
    rootMargin: '200px 0px',
    threshold: 0
  })

  itemRefs.forEach((el) => {
    observer.value?.observe(el)
  })

  await imageList.fetchImages(1, uiStore.imagePageSize)
})

onUnmounted(() => {
  observer.value?.disconnect()
  observer.value = null
})
</script>
