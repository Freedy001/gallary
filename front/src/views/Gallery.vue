<template>
  <AppLayout>
    <template #header>
      <TopBar
        ref="topBarRef"
        :grid-density="uiStore.gridDensity"
        :is-selection-mode="uiStore.isSelectionMode"
        :selected-count="imageStore.selectedCount"
        :show-upload="true"
        :total-count="imageStore.images.length"
        @open-search="uiStore.openCommandPalette"
        @select-all="handleSelectAll"
        @exit-selection="exitSelectionMode"
        @density-change="uiStore.setGridDensity"
        @files-selected="handleFilesSelected"
      >
        <!-- 左侧：搜索模式或默认搜索按钮 -->
        <template #left>
          <!-- 搜索模式下显示搜索状态 -->
          <div v-if="imageStore.isSearchMode" class="flex items-center gap-4 animate-fade-in">
            <div
              class="flex items-center gap-3 rounded-xl bg-linear-to-r from-primary-500/20 to-pink-500/20 px-4 py-2.5 border border-primary-500/30 shadow-[0_0_15px_rgba(139,92,246,0.1)] cursor-pointer hover:bg-white/5 transition-colors group"
              @click="uiStore.openCommandPalette"
            >
              <MagnifyingGlassIcon class="h-4 w-4 text-primary-300" />
              <span :title="imageStore.searchDescription" class="text-sm font-medium text-white truncate max-w-[200px]">
                {{ imageStore.searchDescription }}
              </span>
              <button
                class="ml-2 rounded-full p-1 hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
                title="退出搜索"
                @click.stop="imageStore.exitSearch"
              >
                <XMarkIcon class="h-4 w-4" />
              </button>
            </div>
          </div>
          <!-- 正常模式下显示搜索按钮 -->
          <button
            v-else
            class="group flex w-75 items-center gap-3 rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-gray-400 transition-all hover:border-primary-500/30 hover:bg-white/10 hover:text-white hover:shadow-[0_0_15px_rgba(139,92,246,0.1)]"
            @click="uiStore.openCommandPalette"
          >
            <MagnifyingGlassIcon class="h-5 w-5 text-gray-500 transition-colors group-hover:text-primary-400" />
            <span class="font-light tracking-wide">搜索影像记忆...</span>
            <div class="ml-auto flex gap-1">
              <kbd class="hidden rounded bg-white/10 px-2 py-0.5 text-xs font-mono text-gray-500 group-hover:text-gray-300 md:inline-block">
                {{ isMac ? '⌘' : 'Ctrl' }}
              </kbd>
              <kbd class="hidden rounded bg-white/10 px-2 py-0.5 text-xs font-mono text-gray-500 group-hover:text-gray-300 md:inline-block">
                K
              </kbd>
            </div>
          </button>
        </template>

        <!-- 选择模式下的操作按钮 -->
        <template #selection-actions>
          <button
            v-if="imageStore.selectedCount > 0"
            class="flex items-center gap-2 rounded-xl bg-blue-500/10 border border-blue-500/20 px-5 py-2.5 text-sm font-medium text-blue-400 transition-all hover:bg-blue-500/20 hover:shadow-[0_0_15px_rgba(59,130,246,0.2)]"
            @click="handleBatchDownload"
          >
            <ArrowDownTrayIcon class="h-4 w-4" />
            <span>下载 ({{ imageStore.selectedCount }})</span>
          </button>
          <button
            v-if="imageStore.selectedCount > 0"
            class="flex items-center gap-2 rounded-xl bg-red-500/10 border border-red-500/20 px-5 py-2.5 text-sm font-medium text-red-400 transition-all hover:bg-red-500/20 hover:shadow-[0_0_15px_rgba(239,68,68,0.2)]"
            @click="handleBatchDelete"
          >
            <TrashIcon class="h-4 w-4" />
            <span>删除 ({{ imageStore.selectedCount }})</span>
          </button>
        </template>

        <!-- 正常模式下的操作按钮 -->
        <template #actions>
          <button
            class="rounded-xl px-4 py-2.5 text-sm font-medium text-gray-400 hover:bg-white/5 hover:text-white transition-colors"
            @click="enterSelectionMode"
          >
            选择
          </button>
          <button
            class="relative flex items-center gap-2 rounded-xl bg-white px-5 py-2.5 text-sm font-bold text-black transition-all hover:shadow-[0_0_20px_rgba(255,255,255,0.3)] hover:scale-105 active:scale-95"
            @click="handleUploadClick"
          >
            <ArrowUpTrayIcon class="h-4 w-4" />
            <span>上传</span>
          </button>
        </template>
      </TopBar>
    </template>

    <template #default>
      <!-- 命令面板 -->
      <CommandPalette />
      <!-- 图片网格 -->
      <ImageGrid/>
    </template>

    <template #overlay>
      <!-- 悬浮时间线 -->
      <Timeline />
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {computed, onMounted, ref} from 'vue'
import {useImageStore} from '@/stores/image'
import {useUIStore} from '@/stores/ui'
import {useDialogStore} from '@/stores/dialog'
import AppLayout from '@/components/layout/AppLayout.vue'
import CommandPalette from '@/components/search/CommandPalette.vue'
import ImageGrid from '@/components/gallery/ImageGrid.vue'
import Timeline from '@/components/gallery/Timeline.vue'
import TopBar from '@/components/layout/TopBar.vue'
import type {Image, Pageable} from '@/types'
import {imageApi} from '@/api/image'
import {ArrowDownTrayIcon, ArrowUpTrayIcon, MagnifyingGlassIcon, TrashIcon, XMarkIcon,} from '@heroicons/vue/24/outline'

const imageStore = useImageStore()
const uiStore = useUIStore()
const dialogStore = useDialogStore()
const topBarRef = ref<InstanceType<typeof TopBar> | null>(null)

const isMac = computed(() => navigator.userAgent.includes('Mac'))

// 选择模式
function enterSelectionMode() {
  uiStore.setSelectionMode(true)
}

function exitSelectionMode() {
  uiStore.setSelectionMode(false)
  imageStore.clearSelection()
}

function handleSelectAll() {
  const isAllSelected = imageStore.images.length > 0 && imageStore.selectedCount === imageStore.images.length
  if (isAllSelected) {
    imageStore.clearSelection()
  } else {
    imageStore.images.forEach(img => img && imageStore.selectImage(img.id))
  }
}

// 批量操作
async function handleBatchDelete() {
  const confirmed = await dialogStore.confirm({
    title: '确认删除',
    message: `确定要删除选中的 ${imageStore.selectedCount} 张图片吗？`,
    type: 'warning',
    confirmText: '删除'
  })

  if (!confirmed) return

  try {
    await imageStore.deleteBatch()
    if (imageStore.images.length === 0) {
      exitSelectionMode()
    }
  } catch (error) {
    console.error('批量删除失败', error)
    dialogStore.alert({
      title: '删除失败',
      message: '批量删除过程中发生错误，请重试。',
      type: 'error'
    })
  }
}

function handleBatchDownload() {
  if (imageStore.selectedCount === 0) return
  imageApi.downloadZipped(imageStore.selectedIds)
}

// 上传
function handleUploadClick() {
  topBarRef.value?.triggerUpload()
}

function handleFilesSelected(files: File[]) {
  uiStore.addUploadTask(files)
  uiStore.openUploadDrawer()
}

onMounted(async () => {
  const pageSize = uiStore.imagePageSize
  await imageStore.refreshImages(async (page: number, size: number): Promise<Pageable<Image>> => (await imageApi.getList(page, size)).data, pageSize)

  if (imageStore.images.length === 0) {
    await imageStore.fetchImages(1, pageSize)
  }
})
</script>
