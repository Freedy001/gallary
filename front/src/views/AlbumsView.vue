<template>
  <AppLayout>
    <template #header>
      <div class="relative z-30 flex h-20 w-full items-center justify-between border-b border-white/5 bg-transparent px-8 transition-all duration-300 backdrop-blur-sm">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-white/5 text-primary-400 ring-1 ring-white/10">
            <RectangleStackIcon class="h-5 w-5"/>
          </div>
          <div class="flex flex-col">
            <h1 class="text-lg font-medium text-white leading-tight">相册</h1>
            <span class="text-xs text-gray-500 font-mono mt-0.5">{{ albumStore.total }} 个相册</span>
          </div>
        </div>

        <button
          @click="showCreateModal = true"
          class="flex items-center gap-2 rounded-xl bg-primary-500/10 border border-primary-500/20 px-5 py-2.5 text-sm font-medium text-primary-400 transition-all hover:bg-primary-500/20 hover:shadow-[0_0_15px_rgba(139,92,246,0.2)]"
        >
          <PlusIcon class="h-4 w-4" />
          <span>新建相册</span>
        </button>
      </div>
    </template>

    <template #default>
      <div class="p-6 min-h-[calc(100vh-5rem)]">
        <!-- 加载状态 -->
        <div v-if="albumStore.loading && albumStore.albums.length === 0" class="flex h-64 items-center justify-center">
          <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent shadow-[0_0_15px_rgba(139,92,246,0.5)]"></div>
        </div>

        <!-- 空状态 -->
        <div v-else-if="albumStore.albums.length === 0" class="flex h-64 flex-col items-center justify-center text-gray-500">
          <div class="rounded-2xl bg-white/5 p-4 mb-4 ring-1 ring-white/10">
            <RectangleStackIcon class="h-8 w-8 text-gray-400" />
          </div>
          <p class="text-sm text-gray-400 font-light tracking-wide">暂无相册</p>
          <button
            @click="showCreateModal = true"
            class="mt-4 text-sm text-primary-400 hover:text-primary-300"
          >
            创建第一个相册
          </button>
        </div>

        <!-- 相册网格 -->
        <div v-else class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">
          <div
            v-for="album in albumStore.albums"
            :key="album.id"
            @click="navigateToAlbum(album.id)"
            class="group cursor-pointer"
          >
            <div class="relative aspect-square rounded-2xl overflow-hidden bg-white/5 ring-1 ring-white/10 transition-all duration-300 hover:ring-primary-500/50 hover:shadow-[0_0_30px_rgba(139,92,246,0.2)] hover:scale-[1.02]">
              <!-- 封面图 -->
              <img
                v-if="album.cover_image?.thumbnail_url"
                :src="album.cover_image.thumbnail_url"
                :alt="album.name"
                class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
              />
              <div v-else class="w-full h-full flex items-center justify-center bg-gradient-to-br from-gray-800/50 to-gray-900/50">
                <RectangleStackIcon class="h-12 w-12 text-white/20" />
              </div>

              <!-- 渐变遮罩 -->
              <div class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"></div>

              <!-- 相册信息 -->
              <div class="absolute bottom-0 left-0 right-0 p-4">
                <h3 class="text-white font-medium truncate">{{ album.name }}</h3>
                <p class="text-xs text-gray-400 mt-1">{{ album.image_count }} 张照片</p>
              </div>

              <!-- 删除按钮 -->
              <button
                @click.stop="handleDeleteAlbum(album)"
                class="absolute top-3 right-3 p-2 rounded-lg bg-black/40 text-white/60 opacity-0 group-hover:opacity-100 hover:bg-red-500/80 hover:text-white transition-all"
              >
                <TrashIcon class="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 创建相册弹窗 -->
      <CreateAlbumModal v-model="showCreateModal" @created="onAlbumCreated" />
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAlbumStore } from '@/stores/album'
import { useDialogStore } from '@/stores/dialog'
import AppLayout from '@/components/layout/AppLayout.vue'
import CreateAlbumModal from '@/components/album/CreateAlbumModal.vue'
import { RectangleStackIcon, PlusIcon, TrashIcon } from '@heroicons/vue/24/outline'
import type { Album } from '@/types'

const router = useRouter()
const albumStore = useAlbumStore()
const dialogStore = useDialogStore()
const showCreateModal = ref(false)

function navigateToAlbum(id: number) {
  router.push(`/gallery/albums/${id}`)
}

function onAlbumCreated() {
  showCreateModal.value = false
}

async function handleDeleteAlbum(album: Album) {
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
    await dialogStore.alert({ title: '错误', message: '删除失败', type: 'error' })
  }
}

onMounted(() => {
  albumStore.fetchAlbums()
})
</script>
