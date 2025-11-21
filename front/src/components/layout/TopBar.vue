<template>
  <header class="flex h-16 items-center justify-between border-b border-gray-200 bg-white px-6">
    <!-- 左侧：搜索按钮 -->
    <div v-if="!uiStore.isSelectionMode">
      <button
        @click="uiStore.openCommandPalette"
        class="flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm text-gray-600 transition-colors hover:border-gray-400 hover:bg-gray-50"
      >
        <MagnifyingGlassIcon class="h-4 w-4" />
        <span>搜索图片...</span>
        <kbd class="ml-8 rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-500">
          {{ isMac ? '⌘' : 'Ctrl' }}K
        </kbd>
      </button>
    </div>

    <!-- 选择模式下的左侧 -->
    <div v-else class="flex items-center gap-4">
      <span class="text-lg font-medium text-gray-900">已选择 {{ imageStore.selectedCount }} 项</span>
      <button
        @click="handleSelectAll"
        class="text-sm text-blue-600 hover:text-blue-700"
      >
        {{ isAllSelected ? '取消全选' : '全选' }}
      </button>
    </div>

    <!-- 右侧：上传按钮 + 视图密度滑块 -->
    <div class="flex items-center gap-4">
      <!-- 选择模式下的操作按钮 -->
      <div v-if="uiStore.isSelectionMode" class="flex items-center gap-3">
        <button
          v-if="imageStore.selectedCount > 0"
          @click="handleBatchDelete"
          class="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-100"
        >
          <TrashIcon class="h-4 w-4" />
          <span>删除 ({{ imageStore.selectedCount }})</span>
        </button>
        <button
          @click="exitSelectionMode"
          class="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
        >
          完成
        </button>
      </div>

      <!-- 正常模式下的操作按钮 -->
      <div v-else class="flex items-center gap-3">
        <button
          @click="enterSelectionMode"
          class="rounded-lg px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100"
        >
          选择
        </button>
        <!-- 上传按钮 -->
        <button
          @click="handleUploadClick"
          class="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
        >
          <ArrowUpTrayIcon class="h-4 w-4" />
          <span>上传图片</span>
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
      <div class="h-8 w-px bg-gray-300" />

      <!-- 视图密度滑块 -->
      <div class="flex items-center gap-3">
        <span class="text-sm text-gray-600">密度</span>
        <div class="flex items-center gap-2">
          <Squares2X2Icon class="h-4 w-4 text-gray-500" />
          <input
            type="range"
            min="1"
            max="10"
            :value="uiStore.gridDensity"
            @input="handleDensityChange"
            class="w-32 cursor-pointer accent-blue-600"
          />
          <Square3Stack3DIcon class="h-4 w-4 text-gray-500" />
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
  Squares2X2Icon,
  Square3Stack3DIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'

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

function handleUploadClick() {
  fileInputRef.value?.click()
}

function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files

  if (!files || files.length === 0) return

  // 添加文件到上传队列
  Array.from(files).forEach(file => {
    uiStore.addUploadTask(file)
  })

  // 打开上传抽屉
  if (!uiStore.uploadDrawerOpen) {
    uiStore.openUploadDrawer()
  }

  // 清空input，允许重复选择同一文件
  input.value = ''
}

function handleDensityChange(event: Event) {
  const value = parseInt((event.target as HTMLInputElement).value)
  uiStore.setGridDensity(value)
}
</script>
