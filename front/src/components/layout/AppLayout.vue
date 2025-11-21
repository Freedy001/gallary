<template>
  <div
    class="flex h-screen overflow-hidden bg-white relative"
    @dragenter.prevent="handleDragEnter"
    @dragover.prevent="handleDragOver"
    @dragleave.prevent="handleDragLeave"
    @drop.prevent="handleDrop"
  >
    <!-- 拖拽上传遮罩 -->
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="isDragging"
        class="absolute inset-0 z-50 flex items-center justify-center bg-blue-500/10 backdrop-blur-sm"
      >
        <div class="m-4 flex h-[calc(100%-2rem)] w-[calc(100%-2rem)] flex-col items-center justify-center rounded-2xl border-4 border-dashed border-blue-500 bg-white/50">
          <div class="mb-6 rounded-full bg-white p-6 shadow-xl ring-4 ring-blue-100">
            <ArrowUpTrayIcon class="h-16 w-16 text-blue-600" />
          </div>
          <h3 class="text-3xl font-bold text-gray-900">释放以上传图片</h3>
          <p class="mt-2 text-lg text-gray-600">支持批量上传</p>
        </div>
      </div>
    </Transition>

    <!-- 左侧边栏 -->
    <Sidebar />

    <!-- 主内容区 -->
    <div class="flex flex-1 flex-col overflow-hidden">
      <!-- 顶部栏 -->
      <TopBar />

      <!-- 内容区域 -->
      <div class="flex flex-1 overflow-hidden bg-gray-50 relative">
        <!-- 图片内容滚动区域 -->
        <main id="main-scroll-container" class="flex-1 overflow-y-auto">
          <slot />
        </main>

        <!-- 悬浮层插槽 (用于时间线等) -->
        <slot name="overlay" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useUIStore } from '@/stores/ui'
import { ArrowUpTrayIcon } from '@heroicons/vue/24/outline'
import Sidebar from './Sidebar.vue'
import TopBar from './TopBar.vue'

const uiStore = useUIStore()
const isDragging = ref(false)
const dragCounter = ref(0)

function handleDragEnter(e: DragEvent) {
  dragCounter.value++
  if (e.dataTransfer?.types.includes('Files')) {
    isDragging.value = true
  }
}

function handleDragLeave(e: DragEvent) {
  dragCounter.value--
  if (dragCounter.value === 0) {
    isDragging.value = false
  }
}

function handleDragOver(e: DragEvent) {
  // 必须阻止默认事件才能触发 drop
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'copy'
  }
}

function handleDrop(e: DragEvent) {
  isDragging.value = false
  dragCounter.value = 0

  const files = e.dataTransfer?.files
  if (!files || files.length === 0) return

  let hasImages = false
  Array.from(files).forEach(file => {
    if (file.type.startsWith('image/')) {
      uiStore.addUploadTask(file)
      hasImages = true
    }
  })

  if (hasImages && !uiStore.uploadDrawerOpen) {
    uiStore.openUploadDrawer()
  }
}
</script>
