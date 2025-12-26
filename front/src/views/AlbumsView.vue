<template>
  <AppLayout>
    <template #header>
      <TopBar>
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

        <template #actions>
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
      <div class="p-6 min-h-[calc(100vh-5rem)]">
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

        <!-- 相册网格 -->
        <div v-else :class="gridClass">
          <div
              v-for="album in albumStore.albums"
              :key="album.id"
              @click="navigateToAlbum(album)"
              @contextmenu.prevent="handleContextMenu($event, album)"
              class="group cursor-pointer"
          >
            <div
                class="relative aspect-square rounded-2xl overflow-hidden bg-white/5 ring-1 ring-white/10 transition-all duration-300 hover:ring-primary-500/50 hover:shadow-[0_0_30px_rgba(139,92,246,0.2)] hover:scale-[1.02]">
              <!-- 封面图 -->
              <img
                  v-if="album.cover_image?.thumbnail_url"
                  :src="album.cover_image.thumbnail_url"
                  :alt="album.name"
                  class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
              />
              <div v-else
                   class="w-full h-full flex items-center justify-center bg-gradient-to-br from-gray-800/50 to-gray-900/50">
                <RectangleStackIcon class="h-12 w-12 text-white/20"/>
              </div>

              <!-- 渐变遮罩 -->
              <div class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"></div>

              <!-- 相册信息 -->
              <div class="absolute bottom-0 left-0 right-0 p-4">
                <h3 class="text-white font-medium truncate">{{ album.name }}</h3>
                <p class="text-xs text-gray-400 mt-1">{{ album.image_count }} 张照片</p>
              </div>

              <!-- 菜单按钮 -->
              <button
                  @click.stop="handleContextMenu($event, album)"
                  class="absolute top-3 right-3 p-2 rounded-lg bg-black/40 text-white/60 opacity-0 group-hover:opacity-100 hover:bg-white/20 hover:text-white transition-all"
              >
                <EllipsisHorizontalIcon class="h-5 w-5"/>
              </button>
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

      <!-- 右键菜单 -->
      <ContextMenu v-model="contextMenu.visible" :x="contextMenu.x" :y="contextMenu.y">
        <ContextMenuItem :icon="PencilIcon" @click="handleEditAlbum">
          编辑相册
        </ContextMenuItem>
        <ContextMenuItem :icon="TrashIcon" :danger="true" @click="handleDeleteAlbum">
          删除相册
        </ContextMenuItem>
      </ContextMenu>
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {ref, onMounted, computed} from 'vue'
import {useRouter} from 'vue-router'
import {useAlbumStore} from '@/stores/album'
import {useDialogStore} from '@/stores/dialog'
import {useUIStore} from '@/stores/ui'
import AppLayout from '@/components/layout/AppLayout.vue'
import TopBar from '@/components/layout/TopBar.vue'
import EditAlbumModal from '@/components/album/EditAlbumModal.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import ContextMenuItem from '@/components/common/ContextMenuItem.vue'
import {
  RectangleStackIcon,
  PlusIcon,
  TrashIcon,
  PencilIcon,
  EllipsisHorizontalIcon
} from '@heroicons/vue/24/outline'
import type {Album} from '@/types'

const router = useRouter()
const albumStore = useAlbumStore()
const dialogStore = useDialogStore()
const uiStore = useUIStore()

const showCreateModal = ref(false)
const isEditMode = ref(false)
const editingAlbum = ref<Album | null>(null)

// 右键菜单状态
const contextMenu = ref({visible: false, x: 0, y: 0})
const selectedAlbum = ref<Album | null>(null)

// 根据密度动态计算网格列数，复用 ImageGrid 的逻辑
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
  // 刷新列表
  albumStore.fetchAlbums()
}

function onAlbumUpdated() {
  // 刷新列表
  albumStore.fetchAlbums()
}

function handleContextMenu(event: MouseEvent, album: Album) {
  selectedAlbum.value = album
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY
  }
}

function handleEditAlbum() {
  contextMenu.value.visible = false
  if (selectedAlbum.value) {
    editingAlbum.value = selectedAlbum.value
    isEditMode.value = true
    showCreateModal.value = true
  }
}

async function handleDeleteAlbum() {
  contextMenu.value.visible = false
  const album = selectedAlbum.value
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
    dialogStore.alert({title: '错误', message: '删除失败', type: 'error'})
  }
}

onMounted(() => {
  albumStore.fetchAlbums()
})
</script>
