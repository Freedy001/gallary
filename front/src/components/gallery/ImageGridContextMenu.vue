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

      <ContextMenuItem :icon="(isAltPressed || contextMenuTargetIds.length > 1) ? LinkIcon : ClipboardDocumentIcon" @click="handleCopy">
        {{
          (contextMenuTargetIds.length > 1)
              ? `复制链接 (${contextMenuTargetIds.length})`
              : (isAltPressed ? '复制原图链接' : '复制原图')
        }}
      </ContextMenuItem>
      <ContextMenuItem :icon="RectangleStackIcon" @click="handleAddToAlbum">
        添加到相册 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
      </ContextMenuItem>
      <ContextMenuItem :icon="ShareIcon" @click="handleShare">
        分享 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
      </ContextMenuItem>
      <ContextMenuItem :icon="PencilIcon" @click="handleEdit">
        编辑元数据 {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
      </ContextMenuItem>
      <ContextMenuItem
          :icon="isAltPressed ? ArchiveBoxArrowDownIcon : ArrowDownTrayIcon"
          @click="isAltPressed ? handleBatchDownload() : handleDownload()"
      >
        {{ isAltPressed ? '打包下载' : '下载' }} {{ contextMenuTargetIds.length > 1 ? `(${contextMenuTargetIds.length})` : '' }}
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
import {onMounted, onUnmounted, ref, type Ref} from 'vue'
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
  ClipboardDocumentIcon,
  LinkIcon,
  MinusCircleIcon,
  PencilIcon,
  PhotoIcon,
  RectangleStackIcon,
  ShareIcon,
  TrashIcon,
  XMarkIcon
} from '@heroicons/vue/24/outline'
import ContextMenu from '@/components/common/ContextMenu.vue'
import ContextMenuItem from '@/components/common/ContextMenuItem.vue'
import MetadataEditor from './menu/MetadataEditor.vue'
import CreateShare from '@/components/gallery/menu/CreateShare.vue'
import AddToAlbumModal from '@/components/gallery/menu/AddToAlbumModal.vue'
import {dataSyncService} from "@/services/dataSync.ts";

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

// 键盘状态
const isAltPressed = ref(false)

// 弹窗状态
const isMetadataEditorOpen = ref(false)
const metadataEditorTargetIds = ref<number[]>([])
const metadataEditorInitialData = ref<Image | null>(null)

const isShareModalOpen = ref(false)
const shareTargetIds = ref<number[]>([])

const isAddToAlbumOpen = ref(false)
const addToAlbumTargetIds = ref<number[]>([])

function handleKeyDown(e: KeyboardEvent) {
  if (e.key === 'Alt') isAltPressed.value = true
}

function handleKeyUp(e: KeyboardEvent) {
  if (e.key === 'Alt') isAltPressed.value = false
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
  window.addEventListener('keyup', handleKeyUp)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
  window.removeEventListener('keyup', handleKeyUp)
})

function handleContextMenu(e: MouseEvent, image: Image, index: number) {
  contextMenu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY
  }

  // 检查 Alt 键初始状态
  isAltPressed.value = e.altKey

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
async function handleCopy() {
  closeMenu()
  const ids = contextMenuTargetIds.value
  if (ids.length === 0) return

  // Mode Link if: Alt is pressed OR Multiple images selected
  const isCopyLinkMode = isAltPressed.value || ids.length > 1

  if (isCopyLinkMode) {
    const urls: string[] = []
    
    // Find URLs for all selected IDs
    for (const id of ids) {
      const img = props.images.value.find(i => i?.id === id)
      if (img?.url) {
        urls.push(img.url)
      }
    }

    if (urls.length === 0) {
      dialogStore.alert({title: '错误', message: '未找到可复制的链接', type: 'error'})
      return
    }

    try {
      await navigator.clipboard.writeText(urls.join('\n'))
      
      const msg = ids.length > 1 
        ? `已复制 ${urls.length} 个链接` 
        : '链接已复制'
      
      dialogStore.alert({title: '成功', message: msg, type: 'success'})
    } catch (err) {
      console.error('复制链接失败', err)
      dialogStore.alert({title: '错误', message: '复制链接失败', type: 'error'})
    }
  } else {
    // Single select AND !Alt -> Copy Original Image Blob
    const targetId = ids[0]
    const img = props.images.value.find(i => i?.id === targetId)

    if (!img || !img.url) return

    try {
      // 这里的 url 可能是相对路径或 absolute URL，fetch 应该都能处理
      const blob = await fetch(img.url).then(r => r.blob())
      await navigator.clipboard.write([new ClipboardItem({[blob.type]: blob})])
      dialogStore.alert({title: '成功', message: '图片已复制', type: 'success'})
    } catch (err) {
      console.error('复制图片失败', err)
      dialogStore.alert({title: '错误', message: '复制图片失败', type: 'error'})
    }
  }
}

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

async function handleBatchDownload() {
  closeMenu()
  const ids = contextMenuTargetIds.value.filter((id): id is number => id !== undefined)
  
  if (ids.length === 0) return

  try {
    dialogStore.notify({
      title: '开始下载',
      message: `正在流式打包 ${ids.length} 张图片...`,
      type: 'info',
      duration: 3000
    })

    // 动态导入依赖
    const [{ downloadZip }, streamSaver] = await Promise.all([
      import('client-zip'),
      import('streamsaver')
    ])

    // 创建文件名生成器，处理重复文件名
    const usedNames = new Set<string>()
    const getUniqueFilename = (originalName: string): string => {
      if (!usedNames.has(originalName)) {
        usedNames.add(originalName)
        return originalName
      }

      let counter = 1
      let filename: string
      const lastDot = originalName.lastIndexOf('.')
      
      do {
        if (lastDot > 0) {
          const nameWithoutExt = originalName.substring(0, lastDot)
          const ext = originalName.substring(lastDot)
          filename = `${nameWithoutExt}_${counter}${ext}`
        } else {
          filename = `${originalName}_${counter}`
        }
        counter++
      } while (usedNames.has(filename))
      
      usedNames.add(filename)
      return filename
    }

    // 异步生成器：按需 fetch 图片
    async function* generateFiles() {
      for (const id of ids) {
        const img = props.images.value.find(i => i?.id === id)
        if (!img || !img.url) {
          console.warn(`Skipping image ${id}: not found or no URL`)
          continue
        }

        try {
          const response = await fetch(img.url)
          if (!response.ok) {
            console.error(`Failed to fetch image ${id}: ${response.status}`)
            continue
          }

          yield {
            name: getUniqueFilename(img.original_name),
            lastModified: img.taken_at ? new Date(img.taken_at) : new Date(img.created_at),
            input: response
          }
        } catch (err) {
          console.error(`Error fetching image ${id}:`, err)
        }
      }
    }

    // 生成流式 zip
    const zipStream = downloadZip(generateFiles()).body

    if (!zipStream) {
      throw new Error('Failed to create zip stream')
    }

    // 计算所有图片的总大小
    let totalSize = 0
    for (const id of ids) {
      const img = props.images.value.find(i => i?.id === id)
      if (img?.file_size) {
        totalSize += img.file_size
      }
    }

    // 创建文件写入流（真正的流式下载）
    const fileStream = streamSaver.createWriteStream(
      `images_${new Date().getTime()}.zip`,
      {
        // zip 压缩后的大小通常比原始文件小，但这里提供原始大小作为参考
        // 这样浏览器可以显示下载进度
        size: totalSize > 0 ? totalSize : undefined
      }
    )

    // 将 zip stream 管道连接到文件写入流
    // 这样就实现了：fetch 图片 → 压缩 → 直接写入磁盘
    const writer = fileStream.getWriter()
    const reader = zipStream.getReader()

    try {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        await writer.write(value)
      }
      await writer.close()

      dialogStore.alert({
        title: '成功',
        message: `已完成打包下载`,
        type: 'success'
      })
    } catch (err) {
      await writer.abort()
      throw err
    }
  } catch (err) {
    console.error('Batch download failed:', err)
    dialogStore.alert({
      title: '错误',
      message: '打包下载失败',
      type: 'error'
    })
  }
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
    dataSyncService.emit('images:restored', { ids: contextMenuTargetIds.value, source: 'trash' })
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
