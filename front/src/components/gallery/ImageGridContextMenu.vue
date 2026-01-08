<template>
  <!-- 右键菜单 -->
  <ContextMenu v-model="contextMenu.visible" :x="contextMenu.x" :y="contextMenu.y">
    <template v-if="mode === 'gallery'">
      <template v-if="albumId">
        <!-- 相册模式特有菜单 -->
        <ContextMenuItem v-if="contextMenuTargetIds.length === 1" :icon="PhotoIcon" @click="handleSetAlbumCover">
          设为封面
        </ContextMenuItem>
        <ContextMenuItem :danger="true" :icon="MinusCircleIcon" @click="handleRemoveFromAlbum">
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
      <ContextMenuItem v-if="contextMenuTargetIds.length > 1" :icon="ArchiveBoxArrowDownIcon" @click="handleBatchDownload">
        打包下载 {{ `(${contextMenuTargetIds.length})` }}
      </ContextMenuItem>
      <ContextMenuItem :danger="true" :icon="TrashIcon" @click="handleDelete">
        删除 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
      </ContextMenuItem>
    </template>

    <template v-else-if="mode === 'trash'">
      <ContextMenuItem :icon="ArrowUturnLeftIcon" @click="handleRestore">
        恢复 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
      </ContextMenuItem>
      <ContextMenuItem :danger="true" :icon="XMarkIcon" @click="handlePermanentDelete">
        彻底删除 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
      </ContextMenuItem>
    </template>
  </ContextMenu>

  <!-- 元数据编辑器 -->
  <MetadataEditor
      v-model="isMetadataEditorOpen"
      :image-ids="metadataEditorTargetIds"
      :initial-data="metadataEditorInitialData"
      @saved="onMetadataSaved"
  />

  <!-- 分享弹窗 -->
  <CreateShare
      v-model="isShareModalOpen"
      :selected-count="shareTargetIds.length"
      :selected-ids="shareTargetIds"
  />

  <!-- 添加到相册弹窗 -->
  <AddToAlbumModal
      v-model="isAddToAlbumOpen"
      :exclude-album-id="albumId"
      :image-ids="addToAlbumTargetIds"
      @added="onAddedToAlbum"
  />
</template>

<script lang="ts" setup>
import {ref, type Ref} from 'vue'
import type {Image} from '@/types'
import {useDialogStore} from '@/stores/dialog'
import {useAlbumStore} from '@/stores/album'
import {useUIStore} from '@/stores/ui'
import {imageApi} from '@/api/image'
import {albumApi} from '@/api/album'
import {
  ArchiveBoxArrowDownIcon,
  ArrowDownTrayIcon,
  ArrowUturnLeftIcon,
  MinusCircleIcon,
  PencilIcon,
  PhotoIcon,
  RectangleStackIcon,
  ShareIcon,
  TrashIcon,
  XMarkIcon
} from '@heroicons/vue/24/outline'
import ContextMenu from '@/components/widgets/common/ContextMenu.vue'
import ContextMenuItem from '@/components/widgets/common/ContextMenuItem.vue'
import MetadataEditor from './menu/MetadataEditor.vue'
import CreateShare from '@/components/gallery/menu/CreateShare.vue'
import AddToAlbumModal from '@/components/gallery/menu/AddToAlbumModal.vue'

const props = defineProps<{
  mode: 'gallery' | 'trash'
  albumId?: number
  images: Ref<(Image | null)[]>
  selectedImages: Ref<Set<number>>
  onRemoveImages: (ids: number[]) => void
  onRefresh: () => void
}>()

const dialogStore = useDialogStore()
const albumStore = useAlbumStore()
const uiStore = useUIStore()

// 右键菜单状态
const contextMenu = ref({visible: false, x: 0, y: 0})
const contextMenuTargetIds = ref<number[]>([])
const contextMenuSingleTarget = ref<{ image: Image, index: number } | null>(null)

// 弹窗状态
const isMetadataEditorOpen = ref(false)
const metadataEditorTargetIds = ref<number[]>([])
const metadataEditorInitialData = ref<Image | null>(null)

const isShareModalOpen = ref(false)
const shareTargetIds = ref<number[]>([])

const isAddToAlbumOpen = ref(false)
const addToAlbumTargetIds = ref<number[]>([])

function handleContextMenu(e: MouseEvent, image: Image, index: number) {
  contextMenu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY
  }

  if (props.selectedImages.value.has(image.id)) {
    contextMenuTargetIds.value = Array.from(props.selectedImages.value)
  } else {
    contextMenuTargetIds.value = [image.id]
  }

  contextMenuSingleTarget.value = {image, index}
}

function closeMenu() {
  contextMenu.value.visible = false
}

// ==================== Gallery 模式操作 ====================
async function handleSetAlbumCover() {
  closeMenu()
  if (!props.albumId || contextMenuTargetIds.value.length !== 1) return

  try {
    await albumStore.setAlbumCover(props.albumId, contextMenuTargetIds.value[0] as number)
    dialogStore.alert({title: '成功', message: '设置封面成功', type: 'success'})
  } catch (err) {
    console.error('设置封面失败', err)
    dialogStore.alert({title: '错误', message: '设置封面失败', type: 'error'})
  }
}

async function handleRemoveFromAlbum() {
  closeMenu()
  if (!props.albumId || contextMenuTargetIds.value.length === 0) return

  const ids = contextMenuTargetIds.value

  try {
    await albumApi.removeImages(props.albumId, ids)
    props.onRemoveImages(ids)
    props.selectedImages.value.clear()

    if (albumStore.currentAlbum && albumStore.currentAlbum.id === props.albumId) {
      albumStore.currentAlbum.image_count -= ids.length
    }
  } catch (err) {
    console.error('从相册移除失败', err)
    dialogStore.alert({title: '错误', message: '移除失败', type: 'error'})
  }
}

function handleShare() {
  shareTargetIds.value = contextMenuTargetIds.value
  isShareModalOpen.value = true
  closeMenu()
}

function handleAddToAlbum() {
  addToAlbumTargetIds.value = contextMenuTargetIds.value
  isAddToAlbumOpen.value = true
  closeMenu()
}

function onAddedToAlbum() {
  props.selectedImages.value.clear()
  uiStore.setSelectionMode(false)
}

function handleEdit() {
  metadataEditorTargetIds.value = contextMenuTargetIds.value
  if (contextMenuTargetIds.value.length === 1 && contextMenuSingleTarget.value && contextMenuTargetIds.value[0] === contextMenuSingleTarget.value.image.id) {
    metadataEditorInitialData.value = contextMenuSingleTarget.value.image
  } else if (contextMenuTargetIds.value.length === 1) {
    const img = props.images.value.find(i => i?.id === contextMenuTargetIds.value[0])
    metadataEditorInitialData.value = img || null
  } else {
    metadataEditorInitialData.value = null
  }
  isMetadataEditorOpen.value = true
  closeMenu()
}

function handleDownload() {
  closeMenu()
  for (const targetId of contextMenuTargetIds.value) {
    if (targetId === undefined) continue
    const img = props.images.value.find(i => i?.id === targetId)
    if (img) imageApi.download(targetId, img.original_name)
  }
}

function handleBatchDownload() {
  closeMenu()
  imageApi.downloadZipped(contextMenuTargetIds.value.filter((id): id is number => id !== undefined))
}

async function handleDelete() {
  closeMenu()
  const confirmed = await dialogStore.confirm({
    title: '确认删除',
    message: `确定要删除 ${contextMenuTargetIds.value.length} 张图片吗？`,
    type: 'warning',
    confirmText: '删除'
  })
  if (!confirmed) return

  try {
    await imageApi.deleteBatch(contextMenuTargetIds.value)
    props.onRemoveImages(contextMenuTargetIds.value)
  } catch (e) {
    console.error('Delete failed', e)
  }
}

// ==================== Trash 模式操作 ====================
async function handleRestore() {
  closeMenu()
  try {
    await imageApi.restoreImages(contextMenuTargetIds.value)
    props.onRemoveImages(contextMenuTargetIds.value)
    props.selectedImages.value.clear()
  } catch (err) {
    console.error('恢复图片失败', err)
  }
}

async function handlePermanentDelete() {
  closeMenu()
  const confirmed = await dialogStore.confirm({
    title: '确认彻底删除',
    message: `确定要彻底删除 ${contextMenuTargetIds.value.length} 张图片吗？此操作不可恢复。`,
    type: 'error',
    confirmText: '彻底删除'
  })
  if (!confirmed) return

  try {
    await imageApi.permanentlyDelete(contextMenuTargetIds.value)
    props.onRemoveImages(contextMenuTargetIds.value)
    props.selectedImages.value.clear()
  } catch (err) {
    console.error('彻底删除图片失败', err)
  }
}

function onMetadataSaved() {
  props.onRefresh()
}

// 暴露方法给父组件
defineExpose({
  handleContextMenu,
  isVisible: () => contextMenu.value.visible
})
</script>
