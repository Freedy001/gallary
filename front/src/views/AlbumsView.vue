<template>
  <AppLayout>
    <template #header>
      <TopBar
        :grid-density="uiStore.gridDensity"
        :is-selection-mode="albumStore.hasSelection"
        :selected-count="albumStore.selectedCount"
        :show-density-slider="true"
        :total-count="albumStore.total"
        @select-all="handleSelectAll"
        @exit-selection="albumStore.clearSelection()"
        @density-change="uiStore.setGridDensity"
      >
        <template #left>
          <div class="flex items-center gap-3">
            <div
                class="flex h-10 w-10 items-center justify-center rounded-xl bg-white/5 text-primary-400 ring-1 ring-white/10">
              <RectangleStackIcon class="h-5 w-5"/>
            </div>
            <div class="flex flex-col">
              <h1 class="text-lg font-medium text-white leading-tight">相册</h1>
              <span class="text-xs text-gray-500 font-mono mt-0.5">{{ albumStore.total }} 个相册</span>
            </div>
          </div>
        </template>

        <!-- 选择模式下的操作按钮 -->
        <template #selection-actions>
          <button
              class="flex items-center gap-2 rounded-xl bg-red-500/10 border border-red-500/20 px-5 py-2.5 text-sm font-medium text-red-400 transition-all hover:bg-red-500/20 hover:shadow-[0_0_15px_rgba(239,68,68,0.2)]"
              @click="handleBatchDelete"
          >
            <TrashIcon class="h-4 w-4"/>
            <span>删除 ({{ albumStore.selectedCount }})</span>
          </button>
        </template>

        <!-- 正常模式下的操作按钮 -->
        <template #actions>
          <button
              class="flex items-center gap-2 rounded-xl bg-gradient-to-r from-purple-500/10 to-blue-500/10 border border-purple-500/20 px-5 py-2.5 text-sm font-medium text-purple-400 transition-all hover:from-purple-500/20 hover:to-blue-500/20 hover:shadow-[0_0_15px_rgba(168,85,247,0.2)]"
              @click="showSmartAlbumModal = true"
          >
            <SparklesIcon class="h-4 w-4"/>
            <span>生成智能相册</span>
          </button>
          <button
              @click="openCreateModal"
              class="flex items-center gap-2 rounded-xl bg-primary-500/10 border border-primary-500/20 px-5 py-2.5 text-sm font-medium text-primary-400 transition-all hover:bg-primary-500/20 hover:shadow-[0_0_15px_rgba(139,92,246,0.2)]"
          >
            <PlusIcon class="h-4 w-4"/>
            <span>新建相册</span>
          </button>
        </template>
      </TopBar>
    </template>

    <template #default>
      <div
          ref="containerRef"
          class="relative p-6 min-h-[calc(100vh-5rem)] select-none"
          @mousedown="handleMouseDown"
      >
        <!-- 框选框 -->
        <SelectionBox :style="selectionBoxStyle" />

        <!-- 加载状态 -->
        <div v-if="albumStore.loading && albumStore.albums.length === 0" class="flex h-64 items-center justify-center">
          <div
              class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent shadow-[0_0_15px_rgba(139,92,246,0.5)]"></div>
        </div>

        <!-- 空状态 -->
        <div v-else-if="albumStore.albums.length === 0"
             class="flex h-64 flex-col items-center justify-center text-gray-500">
          <div class="rounded-2xl bg-white/5 p-4 mb-4 ring-1 ring-white/10">
            <RectangleStackIcon class="h-8 w-8 text-gray-400"/>
          </div>
          <p class="text-sm text-gray-400 font-light tracking-wide">暂无相册</p>
          <button
              @click="openCreateModal"
              class="mt-4 text-sm text-primary-400 hover:text-primary-300"
          >
            创建第一个相册
          </button>
        </div>

        <!-- 相册列表 -->
        <div v-else class="space-y-8">
          <!-- 普通相册区域 -->
          <div v-if="normalAlbums.length > 0">
            <div class="flex items-center gap-3 mb-4">
              <RectangleStackIcon class="h-5 w-5 text-gray-400"/>
              <h2 class="text-sm font-medium text-gray-300">普通相册</h2>
              <span class="text-xs text-gray-500">({{ normalAlbums.length }})</span>
            </div>
            <div :class="gridClass">
              <div
                  v-for="(album, index) in normalAlbums"
                  :key="album.id"
                  :ref="(el) => setItemRef(index, el as HTMLElement)"
                  class="group cursor-pointer"
                  @click="handleAlbumClick($event, album)"
                  @contextmenu.prevent="handleContextMenu($event, album)"
              >
                <AlbumCard
                    :album="album"
                    :selected="albumStore.selectedAlbums.has(album.id)"
                    @menu="handleContextMenu($event, album)"
                />
              </div>
            </div>
          </div>

          <!-- 分隔线 -->
          <div v-if="normalAlbums.length > 0 && smartAlbums.length > 0" class="border-t border-white/10"></div>

          <!-- 智能相册区域 -->
          <div v-if="smartAlbums.length > 0">
            <div class="flex items-center gap-3 mb-4">
              <SparklesIcon class="h-5 w-5 text-purple-400"/>
              <h2 class="text-sm font-medium text-gray-300">智能相册</h2>
              <span class="text-xs text-gray-500">({{ smartAlbums.length }})</span>
            </div>
            <div :class="gridClass">
              <div
                  v-for="(album, index) in smartAlbums"
                  :key="album.id"
                  :ref="(el) => setItemRef(normalAlbums.length + index, el as HTMLElement)"
                  class="group cursor-pointer"
                  @click="handleAlbumClick($event, album)"
                  @contextmenu.prevent="handleContextMenu($event, album)"
              >
                <AlbumCard
                    :album="album"
                    :selected="albumStore.selectedAlbums.has(album.id)"
                    :show-probability="true"
                    @menu="handleContextMenu($event, album)"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 创建/编辑相册弹窗 -->
      <EditAlbumModal
        v-model="showCreateModal"
        :edit-mode="isEditMode"
        :initial-data="editingAlbum"
        @created="onAlbumCreated"
        @updated="onAlbumUpdated"
      />

      <!-- 生成智能相册弹窗 -->
      <GenerateSmartAlbumModal
        v-model="showSmartAlbumModal"
        @generated="albumStore.fetchAlbums()"
      />

      <!-- 选择向量模型弹窗 -->
      <SelectModelModal
        v-model="showSelectModelModal"
        @selected="handleModelSelected"
      />

      <!-- 右键菜单 -->
      <ContextMenu v-model="contextMenu.visible" :x="contextMenu.x" :y="contextMenu.y">
        <ContextMenuItem :icon="PencilIcon" @click="handleEditAlbum">
          编辑相册
        </ContextMenuItem>

        <!-- 封面管理 -->
        <div class="h-px bg-white/10 my-1"></div>
        <ContextMenuItem
          v-if="selectedAlbumForMenu?.cover_image_id"
          :icon="XMarkIcon"
          @click="handleRemoveCover"
        >
          移除封面
        </ContextMenuItem>
        <ContextMenuItem :icon="SparklesIcon" @click="handleSetAverageCover">
          设为平均封面
        </ContextMenuItem>

        <div class="h-px bg-white/10 my-1"></div>
        <ContextMenuItem :icon="TrashIcon" :danger="true" @click="handleDeleteAlbum">
          删除相册
        </ContextMenuItem>
      </ContextMenu>
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {computed, onMounted, onUnmounted, ref} from 'vue'
import {useRouter} from 'vue-router'
import {useAlbumStore} from '@/stores/album'
import {useDialogStore} from '@/stores/dialog'
import {useUIStore} from '@/stores/ui'
import {useGenericBoxSelection} from '@/composables/useGenericBoxSelection'
import {albumApi} from '@/api/album'
import AppLayout from '@/components/layout/AppLayout.vue'
import TopBar from '@/components/layout/TopBar.vue'
import EditAlbumModal from '@/components/album/EditAlbumModal.vue'
import GenerateSmartAlbumModal from '@/components/album/GenerateSmartAlbumModal.vue'
import SelectModelModal from '@/components/album/SelectModelModal.vue'
import AlbumCard from '@/components/album/AlbumCard.vue'
import SelectionBox from '@/components/common/SelectionBox.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import ContextMenuItem from '@/components/common/ContextMenuItem.vue'
import {PencilIcon, PlusIcon, RectangleStackIcon, SparklesIcon, TrashIcon, XMarkIcon} from '@heroicons/vue/24/outline'
import type {Album} from '@/types'

const router = useRouter()
const albumStore = useAlbumStore()
const dialogStore = useDialogStore()
const uiStore = useUIStore()

const containerRef = ref<HTMLElement | null>(null)
const showCreateModal = ref(false)
const showSmartAlbumModal = ref(false)
const showSelectModelModal = ref(false)
const isEditMode = ref(false)
const editingAlbum = ref<Album | null>(null)

// 右键菜单状态
const contextMenu = ref({ visible: false, x: 0, y: 0 })
const selectedAlbumForMenu = ref<Album | null>(null)

// 分离普通相册和智能相册
const normalAlbums = computed(() => albumStore.albums.filter(a => !a.is_smart_album))
const smartAlbums = computed(() => albumStore.albums.filter(a => a.is_smart_album))

// 合并所有相册用于框选索引
const allAlbums = computed(() => [...normalAlbums.value, ...smartAlbums.value])

// 框选功能
const itemRefs = new Map<number, HTMLElement>()

function setItemRef(index: number, el: HTMLElement | null) {
  if (el) {
    itemRefs.set(index, el)
  } else {
    itemRefs.delete(index)
  }
}

const { selectionBoxStyle, handleMouseDown, isDragOperation } = useGenericBoxSelection<Album>({
  containerRef,
  itemRefs,
  getItems: () => allAlbums.value,
  getItemId: (album) => album.id,
  toggleSelection: (id) => albumStore.toggleAlbumSelection(id),
  useScroll: false
})

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

  return `grid gap-6 ${mobileClass} ${tabletClass} ${desktopClass}`
})

function handleAlbumClick(event: MouseEvent, album: Album) {
  // 如果是框选操作结束后的点击，忽略
  if (isDragOperation()) return

  // 如果有选中状态，点击切换选中
  if (albumStore.hasSelection || event.ctrlKey || event.metaKey) {
    event.preventDefault()
    albumStore.toggleAlbumSelection(album.id)
    return
  }

  // 正常点击导航到相册详情
  navigateToAlbum(album)
}

function navigateToAlbum(album: Album) {
  albumStore.currentAlbum = album
  router.push(`/gallery/albums/detail`)
}

function openCreateModal() {
  isEditMode.value = false
  editingAlbum.value = null
  showCreateModal.value = true
}

function onAlbumCreated() {
  albumStore.fetchAlbums()
}

function onAlbumUpdated() {
  albumStore.fetchAlbums()
}

function handleContextMenu(event: MouseEvent, album: Album) {
  selectedAlbumForMenu.value = album
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY
  }
}

function handleEditAlbum() {
  contextMenu.value.visible = false
  if (selectedAlbumForMenu.value) {
    editingAlbum.value = selectedAlbumForMenu.value
    isEditMode.value = true
    showCreateModal.value = true
  }
}

async function handleDeleteAlbum() {
  contextMenu.value.visible = false
  const album = selectedAlbumForMenu.value
  if (!album) return

  const confirmed = await dialogStore.confirm({
    title: '确认删除',
    message: `确定要删除相册"${album.name}"吗？相册内的照片不会被删除。`,
    type: 'warning',
    confirmText: '删除'
  })

  if (!confirmed) return

  try {
    await albumStore.deleteAlbum(album.id)
  } catch (err) {
    console.error('删除相册失败', err)
    dialogStore.alert({ title: '错误', message: '删除失败', type: 'error' })
  }
}

function handleSelectAll() {
  albumStore.selectAll()
}

async function handleBatchDelete() {
  const count = albumStore.selectedCount
  const confirmed = await dialogStore.confirm({
    title: '确认批量删除',
    message: `确定要删除选中的 ${count} 个相册吗？相册内的照片不会被删除。`,
    type: 'warning',
    confirmText: '删除'
  })

  if (!confirmed) return

  try {
    await albumStore.deleteSelectedAlbums()
    dialogStore.notify({
      title: '删除成功',
      message: `已删除 ${count} 个相册`,
      type: 'success'
    })
  } catch (err) {
    console.error('批量删除相册失败', err)
    dialogStore.alert({ title: '错误', message: '删除失败', type: 'error' })
  }
}

// 移除封面
async function handleRemoveCover() {
  contextMenu.value.visible = false
  const album = selectedAlbumForMenu.value
  if (!album) return

  try {
    await albumApi.removeCover(album.id)
    dialogStore.notify({
      title: '成功',
      message: '已移除自定义封面，将使用美学评分最高的图片作为封面',
      type: 'success'
    })
    // 刷新相册列表
    await albumStore.fetchAlbums()
  } catch (err) {
    console.error('移除封面失败', err)
    dialogStore.alert({ title: '错误', message: '移除封面失败', type: 'error' })
  }
}

// 设置平均封面
function handleSetAverageCover() {
  contextMenu.value.visible = false
  showSelectModelModal.value = true
}

// 处理模型选择
async function handleModelSelected(modelName: string) {
  const album = selectedAlbumForMenu.value
  if (!album) return

  try {
    await albumApi.setAverageCover(album.id, modelName)
    dialogStore.notify({
      title: '成功',
      message: `已使用 ${modelName} 模型设置最接近平均向量的图片为封面`,
      type: 'success'
    })
    // 刷新相册列表
    await albumStore.fetchAlbums()
  } catch (err: any) {
    console.error('设置平均封面失败', err)
    const errorMsg = err.response?.data?.message || '设置平均封面失败'
    dialogStore.alert({ title: '错误', message: errorMsg, type: 'error' })
  }
}

// 清理选中状态当离开页面
onUnmounted(() => {
  albumStore.clearSelection()
})

onMounted(() => {
  albumStore.fetchAlbums()
})
</script>
