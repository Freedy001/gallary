<template>
  <div
      class="flex h-screen overflow-hidden relative bg-transparent"
      @dragenter.prevent="handleDragEnter"
      @dragover.prevent="handleDragOver"
      @dragleave.prevent="handleDragLeave"
      @drop.prevent="handleDrop"
  >
    <!-- 拖拽上传遮罩 - 极光玻璃态 -->
    <Transition
        enter-active-class="transition duration-500 cubic-bezier(0.16, 1, 0.3, 1)"
        enter-from-class="opacity-0 scale-95 blur-xl"
        enter-to-class="opacity-100 scale-100 blur-0"
        leave-active-class="transition duration-300 ease-in"
        leave-from-class="opacity-100 scale-100 blur-0"
        leave-to-class="opacity-0 scale-95 blur-xl"
    >
      <div
          v-if="isDragging"
          class="absolute inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-xl"
      >
        <div
            class="relative m-4 flex h-[calc(100%-2rem)] w-[calc(100%-2rem)] flex-col items-center justify-center rounded-3xl border border-white/10 bg-white/5 shadow-2xl overflow-hidden">

          <!-- 动态光效背景 -->
          <div class="absolute inset-0 overflow-hidden pointer-events-none">
            <div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[500px] h-[500px] bg-primary-500/20 blur-[120px] rounded-full animate-pulse"></div>
          </div>

          <div class="relative z-10 mb-8 rounded-full bg-white/10 p-8 shadow-[0_0_40px_rgba(139,92,246,0.3)] ring-1 ring-white/20 backdrop-blur-md">
            <ArrowUpTrayIcon class="h-20 w-20 text-primary-100 drop-shadow-lg"/>
          </div>
          <h3 class="relative z-10 text-4xl font-bold text-white tracking-tight drop-shadow-md">释放即刻上传</h3>
          <p class="relative z-10 mt-4 text-lg text-gray-400 font-light">支持批量拖拽 • 自动去重</p>
        </div>
      </div>
    </Transition>

    <!-- 左侧边栏 -->
    <Sidebar/>

    <!-- 主内容区 -->
    <div class="flex flex-1 flex-col overflow-hidden relative z-0">
      <!-- 顶部栏插槽 -->
      <slot name="header"/>

      <!-- 内容区域 -->
      <div class="flex flex-1 overflow-hidden relative">
        <!-- 图片内容滚动区域 -->
        <main id="main-scroll-container" class="flex-1 overflow-y-auto scroll-smooth">
          <slot/>
        </main>

        <!-- 悬浮层插槽 (用于时间线等) -->
        <slot name="overlay"/>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {ref} from 'vue'
import {useUIStore} from '@/stores/ui'
import {useDialogStore} from '@/stores/dialog'
import {ArrowUpTrayIcon} from '@heroicons/vue/24/outline'
import Sidebar from './Sidebar.vue'

const uiStore = useUIStore()
const dialogStore = useDialogStore()
const isDragging = ref(false)
const dragCounter = ref(0)

function handleDragEnter(e: DragEvent) {
  dragCounter.value++
  // 支持文件和目录的拖拽检测
  if (e.dataTransfer?.types.includes('Files') || e.dataTransfer?.items.length) {
    isDragging.value = true
  }
}

function handleDragLeave(_e: DragEvent) {
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

async function handleDrop(e: DragEvent) {
  isDragging.value = false
  dragCounter.value = 0

  const items = e.dataTransfer?.items
  const dataFiles = e.dataTransfer?.files

  if ((!items || items.length === 0) && (!dataFiles || dataFiles.length === 0)) return

  const files: File[] = []

  // 优先使用 webkitGetAsEntry 处理（支持文件夹）
  if (items && items.length > 0) {
    const hasEntry = items[0]?.webkitGetAsEntry
    
    if (hasEntry) {
      // 支持 webkitGetAsEntry，可处理文件夹
      await Promise.all(
        Array.from(items).map(async (item) => {
          if (item.kind === 'file') {
            const entry = item.webkitGetAsEntry()
            if (entry) {
              const collectedFiles = await traverseFileTree(entry)
              files.push(...collectedFiles)
            } else {
              // entry 为 null，回退到 getAsFile
              const file = item.getAsFile()
              if (file) files.push(file)
            }
          }
        })
      )
    } else {
      // 不支持 webkitGetAsEntry，直接获取文件
      Array.from(items).forEach((item) => {
        if (item.kind === 'file') {
          const file = item.getAsFile()
          if (file) files.push(file)
        }
      })
    }
  }

  // 如果 items 方式没有获取到文件，尝试从 files 获取
  if (files.length === 0 && dataFiles && dataFiles.length > 0) {
    files.push(...Array.from(dataFiles))
  }

  // 过滤出图片文件
  const imageFiles = files.filter(file => file.type.startsWith('image/'))
  
  // 提示用户不支持的文件格式
  if (imageFiles.length === 0 && files.length > 0) {
    dialogStore.alert({
      title: '不支持的文件格式',
      message: '仅支持上传图片文件（JPG、PNG、GIF、WebP 等）',
      type: 'warning'
    })
    return
  }
  
  if (imageFiles.length === 0) return

  uiStore.addUploadTask(imageFiles)
  if (!uiStore.uploadDrawerOpen) {
    uiStore.openUploadDrawer()
  }
}

// 递归遍历文件树，支持文件夹
function traverseFileTree(entry: FileSystemEntry): Promise<File[]> {
  return new Promise((resolve) => {
    if (entry.isFile) {
      // 如果是文件，直接返回
      (entry as FileSystemFileEntry).file((file) => {
        resolve([file])
      })
    } else if (entry.isDirectory) {
      // 如果是目录，递归读取
      const dirReader = (entry as FileSystemDirectoryEntry).createReader()
      const files: File[] = []

      const readEntries = () => {
        dirReader.readEntries(async (entries) => {
          if (entries.length === 0) {
            // 读取完毕
            resolve(files)
          } else {
            // 递归处理每个入口
            for (const childEntry of entries) {
              const childFiles = await traverseFileTree(childEntry)
              files.push(...childFiles)
            }
            // 继续读取（目录项可能分批返回）
            readEntries()
          }
        })
      }

      readEntries()
    } else {
      resolve([])
    }
  })
}
</script>
