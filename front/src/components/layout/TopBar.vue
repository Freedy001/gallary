<template>
  <header class="flex h-16 items-center justify-between border-b border-gray-200 bg-white px-6">
    <!-- 左侧：搜索按钮 -->
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

    <!-- 右侧：上传按钮 + 视图密度滑块 -->
    <div class="flex items-center gap-4">
      <!-- 上传按钮 -->
      <button
        @click="handleUploadClick"
        class="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
      >
        <ArrowUpTrayIcon class="h-4 w-4" />
        <span>上传图片</span>
      </button>

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
            max="5"
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
import {
  MagnifyingGlassIcon,
  ArrowUpTrayIcon,
  Squares2X2Icon,
  Square3Stack3DIcon,
} from '@heroicons/vue/24/outline'

const uiStore = useUIStore()
const fileInputRef = ref<HTMLInputElement>()

const isMac = computed(() => {
  return navigator.userAgent.includes('Mac')
})

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
