<template>
  <Modal v-model="isOpen" size="lg" title="智能相册生成结果">
    <!-- 生成结果 -->
    <div v-if="smartAlbumStore.result" class="space-y-6 py-2">
      <!-- 成功状态卡片 -->
      <div class="relative overflow-hidden rounded-2xl bg-white/5 p-6 border border-white/10 shadow-[0_0_30px_rgba(255,255,255,0.05)]">
        <!-- 背景装饰 -->
        <div class="absolute -top-10 -right-10 w-32 h-32 bg-primary-500/10 rounded-full blur-3xl"></div>
        <div class="absolute -bottom-10 -left-10 w-24 h-24 bg-blue-500/10 rounded-full blur-2xl"></div>

        <div class="relative flex items-start gap-4">
          <div class="p-3 rounded-full bg-primary-500/20 text-primary-400 shadow-[0_0_15px_rgba(139,92,246,0.3)]">
            <CheckCircleIcon class="h-8 w-8"/>
          </div>
          <div class="flex-1 space-y-2">
            <h3 class="text-xl font-semibold text-gray-100">生成完成</h3>
            <div class="flex flex-wrap gap-x-8 gap-y-2 text-sm text-gray-300/80">
              <div class="flex items-center gap-2">
                <span class="text-2xl font-bold text-white tabular-nums tracking-tight">{{ smartAlbumStore.result.cluster_count }}</span>
                <span>个智能相册</span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-2xl font-bold text-white tabular-nums tracking-tight">{{ smartAlbumStore.result.total_images }}</span>
                <span>张处理图片</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 噪声图片展示 -->
      <div v-if="smartAlbumStore.result.noise_count && smartAlbumStore.result.noise_count > 0"
           class="rounded-2xl border border-amber-500/20 bg-amber-500/5 overflow-hidden">
        <div class="px-6 py-4">
          <div class="flex items-center gap-3">
            <div class="p-2 rounded-lg bg-amber-500/20 text-amber-400">
              <ExclamationTriangleIcon class="h-5 w-5"/>
            </div>
            <div>
              <p class="font-medium text-amber-200">未归类图片 (噪声点)</p>
              <p class="text-xs text-amber-500/80 mt-0.5">{{ smartAlbumStore.result.noise_count }} 张图片特征不明显，未被归入任何相册</p>
            </div>
          </div>
        </div>

        <div class="border-t border-amber-500/10 bg-black/20">
          <div v-if="loadingNoiseImages" class="flex items-center justify-center py-12">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-amber-500"></div>
          </div>
          <div v-else-if="noiseImages.length > 0"
               class="grid grid-cols-5 gap-3 max-h-[400px] overflow-y-auto p-5 scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent">
            <div
                v-for="(image, index) in noiseImages"
                :key="image.id"
                :class="{ 'ring-2 ring-primary-500': selectedNoiseImageIds.has(image.id) }"
                class="relative aspect-square rounded-xl overflow-hidden group cursor-pointer bg-gray-800/50 shadow-lg ring-1 ring-white/10 hover:ring-amber-500/50 hover:shadow-[0_0_15px_rgba(245,158,11,0.2)] transition-all duration-300"
                @click.stop="handleNoiseImageClick(image.id)"
                @dblclick.stop="handleImageDoubleClick(index)"
                @contextmenu.prevent.stop="handleNoiseImageContextMenu($event, image.id)"
            >
              <img
                  :alt="image.original_name"
                  :src="image.thumbnail_url || image.url"
                  class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
                  loading="lazy"
              />
              <!-- 遮罩 -->
              <div class="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors duration-300"></div>
              <!-- 选中标记 -->
              <div v-if="selectedNoiseImageIds.has(image.id)" class="absolute top-2 right-2 bg-primary-500 rounded-full p-1">
                <CheckIcon class="h-3 w-3 text-white"/>
              </div>
            </div>
          </div>
          <p v-else class="text-sm text-gray-500 text-center py-8">无法加载噪声图片</p>
        </div>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="smartAlbumStore.errorMessage" class="p-4 bg-red-500/10 border border-red-500/20 rounded-xl">
      <div class="flex items-center gap-2 text-red-400 mb-2">
        <XCircleIcon class="h-5 w-5"/>
        <span class="font-medium">生成失败</span>
      </div>
      <p class="text-sm text-gray-300">{{ smartAlbumStore.errorMessage }}</p>
    </div>

    <!-- 操作按钮 -->
    <div class="flex justify-end gap-3 pt-4">
      <button
          class="px-5 py-2.5 rounded-xl bg-white/10 text-white hover:bg-white/20 transition-colors"
          type="button"
          @click="handleClose"
      >
        确定
      </button>
    </div>
  </Modal>

  <!-- 图片预览器 -->
  <ImageViewer
      v-model:index="viewerIndex"
      :images="noiseImages"
      @delete="handleViewerDelete"
  />

  <!-- 右键菜单 -->
  <ContextMenu v-model="contextMenu.visible" :x="contextMenu.x" :y="contextMenu.y">
    <ContextMenuItem :icon="EyeIcon" @click="handleView">
      查看
    </ContextMenuItem>
    <ContextMenuItem :icon="RectangleStackIcon" @click="handleAddToAlbum">
      添加到相册 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
    </ContextMenuItem>
  </ContextMenu>

  <!-- 添加到相册弹窗 -->
  <AddToAlbumModal
      v-model="isAddToAlbumOpen"
      :image-ids="addToAlbumTargetIds"
      :include-smart-album="true"
      @added="onAddedToAlbum"
  />
</template>

<script lang="ts" setup>
import {computed, ref, watch} from 'vue'
import {
  CheckCircleIcon,
  CheckIcon,
  ExclamationTriangleIcon,
  EyeIcon,
  RectangleStackIcon,
  XCircleIcon
} from '@heroicons/vue/24/outline'
import Modal from '@/components/common/Modal.vue'
import ImageViewer from '@/components/gallery/ImageViewer.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import ContextMenuItem from '@/components/common/ContextMenuItem.vue'
import AddToAlbumModal from '@/components/gallery/menu/AddToAlbumModal.vue'
import {useSmartAlbumStore} from '@/stores/smartAlbum'
import {useDialogStore} from '@/stores/dialog'
import {imageApi} from '@/api/image'
import type {Image} from '@/types'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const smartAlbumStore = useSmartAlbumStore()
const dialogStore = useDialogStore()

const isOpen = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const loadingNoiseImages = ref(false)
const noiseImages = ref<Image[]>([])
const viewerIndex = ref(-1)

// 右键菜单状态
const contextMenu = ref({ visible: false, x: 0, y: 0 })
const contextMenuTargetIds = ref<number[]>([])
const contextMenuSingleTarget = ref<{ imageId: number, index: number } | null>(null)
const selectedNoiseImageIds = ref<Set<number>>(new Set())

// 添加到相册弹窗状态
const isAddToAlbumOpen = ref(false)
const addToAlbumTargetIds = ref<number[]>([])

// 加载噪声图片
async function loadNoiseImages() {
  const ids = smartAlbumStore.result?.noise_image_ids
  if (!ids || ids.length === 0) return

  try {
    loadingNoiseImages.value = true
    const res = await imageApi.getByIds(ids)
    noiseImages.value = res.data || []
  } catch (err) {
    console.error('加载噪声图片失败', err)
  } finally {
    loadingNoiseImages.value = false
  }
}

// 点击噪声图片（选择/取消选择）
function handleNoiseImageClick(imageId: number) {
  if (selectedNoiseImageIds.value.has(imageId)) {
    selectedNoiseImageIds.value.delete(imageId)
  } else {
    selectedNoiseImageIds.value.add(imageId)
  }
  // 触发响应式更新
  selectedNoiseImageIds.value = new Set(selectedNoiseImageIds.value)
}

// 右键菜单
function handleNoiseImageContextMenu(e: MouseEvent, imageId: number) {
  e.stopPropagation()
  
  // 查找图片索引
  const index = noiseImages.value.findIndex(img => img.id === imageId)
  contextMenuSingleTarget.value = { imageId, index }
  
  contextMenu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY
  }

  // 如果右键点击的图片已被选中，则对所有选中的图片操作
  if (selectedNoiseImageIds.value.has(imageId)) {
    contextMenuTargetIds.value = Array.from(selectedNoiseImageIds.value)
  } else {
    contextMenuTargetIds.value = [imageId]
  }
}

// 查看图片
function handleView() {
  contextMenu.value.visible = false
  if (contextMenuSingleTarget.value) {
    viewerIndex.value = contextMenuSingleTarget.value.index
  }
}

// 添加到相册
function handleAddToAlbum() {
  contextMenu.value.visible = false
  addToAlbumTargetIds.value = contextMenuTargetIds.value
  isAddToAlbumOpen.value = true
}

// 添加到相册成功回调
function onAddedToAlbum() {
  selectedNoiseImageIds.value.clear()
}

// 双击图片打开预览
function handleImageDoubleClick(index: number) {
  viewerIndex.value = index
}

// 处理查看器中的删除操作
function handleViewerDelete(id: number) {
  // 从噪声图片列表中移除
  noiseImages.value = noiseImages.value.filter(img => img.id !== id)
  
  // 更新 store 中的结果数据
  if (smartAlbumStore.result) {
    smartAlbumStore.result.noise_image_ids = smartAlbumStore.result.noise_image_ids?.filter(imgId => imgId !== id) || []
    smartAlbumStore.result.noise_count = smartAlbumStore.result.noise_image_ids.length
  }
  
  // 调整查看器索引
  if (viewerIndex.value >= noiseImages.value.length) {
    viewerIndex.value = noiseImages.value.length - 1
  }
  if (noiseImages.value.length === 0) {
    viewerIndex.value = -1
  }
}

// 监听弹窗打开/关闭，重置状态
watch(isOpen, async (val) => {
  if (val) {
    // 打开时自动加载噪声图片
    if (smartAlbumStore.result?.noise_image_ids?.length && noiseImages.value.length === 0) {
      await loadNoiseImages()
    }
  } else {
    noiseImages.value = []
    viewerIndex.value = -1
    selectedNoiseImageIds.value.clear()
    contextMenu.value.visible = false
  }
})

async function handleClose() {
  // 提示用户将清除结果
  const confirmed = await dialogStore.confirm({
    title: '确认关闭',
    message: '关闭后将清除当前的智能相册生成结果，确定要关闭吗？',
    type: 'warning',
    confirmText: '确定',
    cancelText: '取消'
  })
  
  if (!confirmed) return
  
  isOpen.value = false
  // 关闭弹窗时，清除结果状态，这样 Sidebar 上的提示也会消失
  smartAlbumStore.resetState()
}
</script>
