<template>
  <Modal v-model="isOpen" title="添加到相册" size="sm">
    <div class="space-y-4">
      <!-- 图片数量提示 -->
      <div class="flex items-center gap-2 text-sm text-gray-400">
        <PhotoIcon class="h-4 w-4" />
        <span>已选择 {{ imageIds.length }} 张图片</span>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <!-- 相册列表 -->
      <div v-else-if="filteredAlbums.length > 0" class="max-h-64 overflow-y-auto space-y-2 scrollbar-thin">
        <button
          v-for="album in filteredAlbums"
          :key="album.id"
          @click="toggleAlbum(album.id)"
          class="w-full flex items-center gap-3 rounded-xl px-4 py-3 transition-all"
          :class="[
            selectedAlbumIds.has(album.id)
              ? 'bg-primary-500/20 ring-1 ring-primary-500/50'
              : 'bg-white/5 hover:bg-white/10'
          ]"
        >
          <!-- 封面缩略图 -->
          <div class="h-10 w-10 rounded-lg overflow-hidden bg-white/5 flex-shrink-0">
            <img
              v-if="album.cover_image?.thumbnail_url"
              :src="album.cover_image.thumbnail_url"
              :alt="album.name"
              class="h-full w-full object-cover"
            />
            <div v-else class="h-full w-full flex items-center justify-center">
              <RectangleStackIcon class="h-5 w-5 text-white/20" />
            </div>
          </div>

          <!-- 相册信息 -->
          <div class="flex-1 text-left min-w-0">
            <div class="text-sm font-medium text-white truncate">{{ album.name }}</div>
            <div class="text-xs text-gray-500">{{ album.image_count }} 张照片</div>
          </div>

          <!-- 选中状态 -->
          <div
            class="flex h-5 w-5 items-center justify-center rounded-full border-2 transition-colors flex-shrink-0"
            :class="[
              selectedAlbumIds.has(album.id)
                ? 'border-primary-500 bg-primary-500 text-white'
                : 'border-white/30'
            ]"
          >
            <CheckIcon v-if="selectedAlbumIds.has(album.id)" class="h-3 w-3" />
          </div>
        </button>
      </div>

      <!-- 空状态 -->
      <div v-else class="py-8 text-center">
        <RectangleStackIcon class="mx-auto h-8 w-8 text-gray-500" />
        <p class="mt-2 text-sm text-gray-400">暂无相册</p>
        <button
          @click="showCreateAlbum = true"
          class="mt-3 text-sm text-primary-400 hover:text-primary-300"
        >
          创建新相册
        </button>
      </div>

      <!-- 新建相册入口 -->
      <button
        v-if="filteredAlbums.length > 0"
        @click="showCreateAlbum = true"
        class="w-full flex items-center justify-center gap-2 rounded-xl border border-dashed border-white/20 px-4 py-3 text-sm text-gray-400 hover:border-primary-500/50 hover:text-primary-400 transition-colors"
      >
        <PlusIcon class="h-4 w-4" />
        <span>新建相册</span>
      </button>

      <!-- 操作按钮 -->
      <div class="flex justify-end gap-3 pt-2">
        <button
          type="button"
          @click="isOpen = false"
          class="px-5 py-2.5 rounded-xl border border-white/10 text-gray-400 hover:bg-white/5 transition-colors"
        >
          取消
        </button>
        <button
          @click="handleConfirm"
          :disabled="selectedAlbumIds.size === 0 || submitting"
          class="px-5 py-2.5 rounded-xl bg-primary-500 text-white hover:bg-primary-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ submitting ? '添加中...' : `添加到 ${selectedAlbumIds.size} 个相册` }}
        </button>
      </div>
    </div>

    <!-- 快速创建相册弹窗 -->
    <EditAlbumModal v-model="showCreateAlbum" @created="onAlbumCreated" />
  </Modal>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {albumApi} from '@/api/album.ts'
import type {Album} from '@/types'
import Modal from '@/components/widgets/common/Modal.vue'
import EditAlbumModal from '../../album/EditAlbumModal.vue'
import {CheckIcon, PhotoIcon, PlusIcon, RectangleStackIcon} from '@heroicons/vue/24/outline'
import {useDialogStore} from "@/stores/dialog.ts";

const dialogStore = useDialogStore();

const props = defineProps<{
  imageIds: number[]
  excludeAlbumId?: number  // 排除的相册ID（当前相册）
}>()

const isOpen = defineModel<boolean>({ default: false })
const emit = defineEmits<{
  added: [albumIds: number[]]
}>()

const albums = ref<Album[]>([])
const loading = ref(false)
const submitting = ref(false)
const selectedAlbumIds = ref<Set<number>>(new Set())
const showCreateAlbum = ref(false)

// 过滤后的相册列表（排除当前相册）
const filteredAlbums = computed(() => {
  if (props.excludeAlbumId) {
    return albums.value.filter(a => a.id !== props.excludeAlbumId)
  }
  return albums.value
})

// 加载相册列表
async function loadAlbums() {
  try {
    loading.value = true
    // 只获取普通相册，排除智能相册
    const { data } = await albumApi.getList({ page: 1, pageSize: 100, isSmart: false })
    albums.value = data.list
  } catch (err) {
    dialogStore.alert({ title: '错误', message: '加载相册列表失败', type: 'error' })
    console.error('加载相册列表失败', err)
  } finally {
    loading.value = false
  }
}

// 切换相册选择
function toggleAlbum(albumId: number) {
  if (selectedAlbumIds.value.has(albumId)) {
    selectedAlbumIds.value.delete(albumId)
  } else {
    selectedAlbumIds.value.add(albumId)
  }
  // 触发响应式更新
  selectedAlbumIds.value = new Set(selectedAlbumIds.value)
}

// 确认添加
async function handleConfirm() {
  if (selectedAlbumIds.value.size === 0 || submitting.value) return

  try {
    submitting.value = true
    const albumIdList = Array.from(selectedAlbumIds.value)

    // 并行添加到所有选中的相册
    await Promise.all(
      albumIdList.map(albumId => albumApi.addImages(albumId, props.imageIds))
    )

    // 更新相册图片数量（本地更新）
    albums.value = albums.value.map(album => {
      if (selectedAlbumIds.value.has(album.id)) {
        return { ...album, image_count: album.image_count + props.imageIds.length }
      }
      return album
    })

    // 获取选中的相册名称
    const selectedAlbumNames = albums.value
      .filter(a => selectedAlbumIds.value.has(a.id))
      .map(a => a.name)
      .join('、')

    // 显示成功通知
    dialogStore.alert({
      title: '添加成功',
      message: `已将 ${props.imageIds.length} 张图片添加到（${selectedAlbumNames}）`,
      type: 'success'
    })

    emit('added', albumIdList)
    isOpen.value = false
  } catch (err) {
    dialogStore.alert({ title: '错误', message: '添加到相册失败', type: 'error' })
    console.error('添加到相册失败', err)
  } finally {
    submitting.value = false
  }
}

// 相册创建成功回调
function onAlbumCreated() {
  showCreateAlbum.value = false
  loadAlbums() // 重新加载相册列表
}

// 监听弹窗打开/关闭
watch(isOpen, (val) => {
  if (val) {
    loadAlbums()
    selectedAlbumIds.value = new Set()
  }
})
</script>
