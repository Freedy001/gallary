<template>
  <header class="relative z-30 flex h-20 w-full items-center justify-between border-b border-white/5 bg-transparent px-8 transition-all duration-300 backdrop-blur-sm">
    <!-- 左侧：搜索按钮 -->
    <div v-if="!uiStore.isSelectionMode" class="w-96">
      <button
        @click="uiStore.openCommandPalette"
        class="group flex w-full items-center gap-3 rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-gray-400 transition-all hover:border-primary-500/30 hover:bg-white/10 hover:text-white hover:shadow-[0_0_15px_rgba(139,92,246,0.1)]"
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
    </div>

    <!-- 选择模式下的左侧 -->
    <div v-else class="flex items-center gap-6 animate-fade-in">
      <div class="flex items-center gap-3 rounded-lg bg-primary-500/10 px-4 py-2 border border-primary-500/20">
        <span class="h-2 w-2 rounded-full bg-primary-500 animate-pulse"></span>
        <span class="text-lg font-medium text-white">已选择 {{ imageStore.selectedCount }} 项</span>
      </div>
      <button
        @click="handleSelectAll"
        class="text-sm font-medium text-primary-400 hover:text-primary-300 hover:underline underline-offset-4"
      >
        {{ isAllSelected ? '取消全选' : '全选所有' }}
      </button>
    </div>

    <!-- 右侧：上传按钮 + 视图密度滑块 -->
    <div class="flex items-center gap-6">
      <!-- 选择模式下的操作按钮 -->
      <div v-if="uiStore.isSelectionMode" class="flex items-center gap-4">
        <button
          v-if="imageStore.selectedCount > 0"
          @click="handleBatchDownload"
          class="flex items-center gap-2 rounded-xl bg-blue-500/10 border border-blue-500/20 px-5 py-2.5 text-sm font-medium text-blue-400 transition-all hover:bg-blue-500/20 hover:shadow-[0_0_15px_rgba(59,130,246,0.2)]"
        >
          <ArrowDownTrayIcon class="h-4 w-4" />
          <span>下载 ({{ imageStore.selectedCount }})</span>
        </button>
        <button
          v-if="imageStore.selectedCount > 0"
          @click="handleBatchDelete"
          class="flex items-center gap-2 rounded-xl bg-red-500/10 border border-red-500/20 px-5 py-2.5 text-sm font-medium text-red-400 transition-all hover:bg-red-500/20 hover:shadow-[0_0_15px_rgba(239,68,68,0.2)]"
        >
          <TrashIcon class="h-4 w-4" />
          <span>删除 ({{ imageStore.selectedCount }})</span>
        </button>
        <button
          @click="exitSelectionMode"
          class="rounded-xl border border-white/10 bg-white/5 px-6 py-2.5 text-sm font-medium text-white hover:bg-white/10 transition-colors"
        >
          完成
        </button>
      </div>

      <!-- 正常模式下的操作按钮 -->
      <div v-else class="flex items-center gap-4">
        <button
          @click="enterSelectionMode"
          class="rounded-xl px-4 py-2.5 text-sm font-medium text-gray-400 hover:bg-white/5 hover:text-white transition-colors"
        >
          选择
        </button>
        <!-- 上传按钮 -->
        <button
          @click="handleUploadClick"
          class="relative flex items-center gap-2 rounded-xl bg-white px-5 py-2.5 text-sm font-bold text-black transition-all hover:shadow-[0_0_20px_rgba(255,255,255,0.3)] hover:scale-105 active:scale-95"
        >
          <ArrowUpTrayIcon class="h-4 w-4" />
          <span>上传</span>
        </button>
      </div>

      <input
        ref="fileInputRef"
        type="file"
        multiple
        accept="image/*"
        class="hidden"
        @change="handleFileSelect"
      />

      <!-- 分隔线 -->
      <div class="h-8 w-px bg-white/10" />

      <!-- 视图密度滑块 -->
      <div class="flex items-center gap-3 group">
        <div class="flex items-center gap-2 px-2 py-1 rounded-lg group-hover:bg-white/5 transition-colors">
          <Squares2X2Icon class="h-4 w-4 text-gray-500 group-hover:text-gray-300" />
          <input
            type="range"
            min="1"
            max="10"
            :value="uiStore.gridDensity"
            @input="handleDensityChange"
            class="w-24 cursor-pointer accent-white h-1 bg-white/10 rounded-full appearance-none hover:bg-white/20"
          />
          <Square3Stack3DIcon class="h-4 w-4 text-gray-500 group-hover:text-gray-300" />
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useUIStore } from '@/stores/ui'
import { useImageStore } from '@/stores/image'
import {
  MagnifyingGlassIcon,
  ArrowUpTrayIcon,
  ArrowDownTrayIcon,
  Squares2X2Icon,
  Square3Stack3DIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'
import { imageApi } from '@/api/image'

const uiStore = useUIStore()
const imageStore = useImageStore()
const fileInputRef = ref<HTMLInputElement>()

const isMac = computed(() => {
  return navigator.userAgent.includes('Mac')
})

const isAllSelected = computed(() => {
  return imageStore.images.length > 0 && imageStore.selectedCount === imageStore.images.length
})

function enterSelectionMode() {
  uiStore.setSelectionMode(true)
}

function exitSelectionMode() {
  uiStore.setSelectionMode(false)
  imageStore.clearSelection()
}

function handleSelectAll() {
  if (isAllSelected.value) {
    imageStore.clearSelection()
  } else {
    imageStore.images.forEach(img => imageStore.selectImage(img.id))
  }
}

async function handleBatchDelete() {
  if (!confirm(`确定要删除选中的 ${imageStore.selectedCount} 张图片吗？`)) return

  try {
    await imageStore.deleteBatch()
    // 如果删除后没有图片了或没有选中了，退出选择模式
    if (imageStore.images.length === 0) {
      exitSelectionMode()
    }
  } catch (error) {
    console.error('批量删除失败', error)
    alert('删除失败')
  }
}

async function handleBatchDownload() {
  if (imageStore.selectedCount === 0) return

  imageApi.downloadZipped(imageStore.selectedIds)
}

function handleUploadClick() {
  fileInputRef.value?.click()
}

function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files

  if (!files || files.length === 0) return

  // 添加文件到上传队列
  uiStore.addUploadTask(Array.from(files))

  // 打开上传抽屉
  uiStore.openUploadDrawer()

  // 清空input，允许重复选择同一文件
  input.value = ''
}

function handleDensityChange(event: Event) {
  const value = parseInt((event.target as HTMLInputElement).value)
  uiStore.setGridDensity(value)
}
</script>
