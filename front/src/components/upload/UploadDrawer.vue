<template>
  <!-- 收起状态的进度条 (嵌入在 Sidebar 中) -->
  <div
      v-if="uiStore.uploadTasks.length > 0"
      class="mb-2"
  >
    <div
        class="flex cursor-pointer items-center gap-3 rounded-lg transition-all hover:bg-gray-50"
        :class="[
        props.collapsed ? 'justify-center border-0 p-1' : 'border border-gray-200 bg-white p-2 shadow-sm px-3'
      ]"
        @click="toggleDrawer"
    >
      <!-- 环形进度条 -->
      <div class="relative flex h-8 w-8 flex-shrink-0 items-center justify-center">
        <svg class="h-full w-full -rotate-90 transform" viewBox="0 0 36 36">
          <!-- 背景圆 -->
          <path
              class="text-gray-100"
              d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
          />
          <!-- 进度圆 -->
          <path
              class="text-blue-600 transition-all duration-300 ease-out"
              :stroke-dasharray="`${uiStore.totalProgress}, 100`"
              d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
              stroke-linecap="round"
          />
        </svg>
        <ArrowUpTrayIcon class="absolute h-4 w-4 text-blue-600"/>
      </div>

      <div v-if="!props.collapsed" class="mr-1 flex flex-1 flex-col min-w-0">
        <span class="text-sm font-bold text-gray-900 truncate">
          {{ uiStore.uploadingCount > 0 ? '正在上传...' : '上传完成' }}
        </span>
        <span class="text-xs font-medium text-gray-500 truncate">
          <span v-if="uiStore.failedCount>0">
               ( {{ uiStore.completedCount }} + <span class="text-red-600">{{
              uiStore.failedCount
            }}</span> )
          </span>
          <span v-else>
          {{ uiStore.completedCount }}
          </span>
          / {{ uiStore.uploadTasks.length }}
        </span>
      </div>
    </div>
  </div>

  <!-- 展开状态的详情框 (左下角 Popover，定位在 Sidebar 旁) -->
  <Teleport to="body">
    <Transition
        enter-active-class="transition duration-300 ease-out"
        enter-from-class="transform scale-95 opacity-0 translate-y-4 -translate-x-4"
        enter-to-class="transform scale-100 opacity-100 translate-y-0 translate-x-0"
        leave-active-class="transition duration-200 ease-in"
        leave-from-class="transform scale-100 opacity-100 translate-y-0 translate-x-0"
        leave-to-class="transform scale-95 opacity-0 translate-y-4 -translate-x-4"
    >
      <div
          v-if="uiStore.uploadDrawerOpen"
          class="fixed z-50 flex max-h-[600px] w-80 flex-col overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-2xl origin-bottom-left"
          :class="[props.collapsed ? 'left-20 bottom-4' : 'left-68 bottom-4']"
          :style="{ left: props.collapsed ? '4.5rem' : '16.5rem' }"
      >
        <!-- 头部 -->
        <div class="flex items-center justify-between border-b border-gray-100 bg-gray-50/50 p-3 backdrop-blur-sm">
          <div class="flex items-center gap-2">
            <div class="flex h-7 w-7 items-center justify-center rounded-full bg-blue-100 text-blue-600">
              <ArrowUpTrayIcon class="h-3.5 w-3.5"/>
            </div>
            <div>
              <h3 class="text-sm font-bold text-gray-900">上传队列</h3>
              <p class="text-xs text-gray-500">{{ uiStore.uploadTasks.length }} 个任务</p>
            </div>
          </div>

          <div class="flex items-center gap-1">
            <button
                v-if="uiStore.completedCount > 0 || uiStore.failedCount > 0"
                @click="uiStore.clearCompletedTasks"
                class="rounded-lg px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-200/50"
            >
              清除
            </button>
            <button
                @click="uiStore.closeUploadDrawer"
                class="rounded-full p-1.5 text-gray-400 hover:bg-gray-200/50 hover:text-gray-600"
            >
              <XMarkIcon class="h-4 w-4"/>
            </button>
          </div>
        </div>

        <!-- 总体进度 -->
        <div v-if="uiStore.uploadingCount > 0" class="border-b border-gray-100 bg-white px-3 py-2">
          <div class="mb-1.5 flex items-center justify-between text-xs">
            <span class="font-medium text-gray-600">总体进度</span>
            <span class="font-bold text-blue-600">{{ uiStore.totalProgress }}%</span>
          </div>
          <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-100">
            <div
                class="h-full bg-blue-500 transition-all duration-300 ease-out"
                :style="{ width: `${uiStore.totalProgress}%` }"
            />
          </div>
        </div>

        <!-- 任务列表 -->
        <div class="flex-1 overflow-y-auto p-2">
          <div class="space-y-2">
            <div
                v-for="task in uiStore.uploadTasks"
                :key="task.id"
                class="group flex items-center gap-3 rounded-xl p-2 transition-colors hover:bg-gray-50"
            >
              <!-- 缩略图 -->
              <div
                  class="relative h-10 w-10 flex-shrink-0 overflow-hidden rounded-lg bg-gray-100 ring-1 ring-gray-900/5">
                <img
                    v-if="task.imageUrl"
                    :src="task.imageUrl"
                    :alt="task.file.name"
                    class="h-full w-full object-cover"
                />
                <div v-else class="flex h-full items-center justify-center">
                  <PhotoIcon class="h-5 w-5 text-gray-400"/>
                </div>

                <!-- 状态覆盖层 -->
                <div
                    v-if="task.status !== 'pending'"
                    class="absolute inset-0 flex items-center justify-center bg-black/20 transition-opacity"
                    :class="{'bg-red-500/20': task.status === 'error'}"
                >
                  <CheckCircleIcon v-if="task.status === 'success'" class="h-5 w-5 text-green-400 drop-shadow-md"/>
                  <ExclamationCircleIcon v-else-if="task.status === 'error'"
                                         class="h-5 w-5 text-red-500 drop-shadow-md"/>
                  <div v-else-if="task.status === 'uploading'"
                       class="h-4 w-4 rounded-full border-2 border-white border-t-transparent animate-spin"/>
                </div>
              </div>

              <!-- 信息 -->
              <div class="flex-1 min-w-0 py-0.5">
                <div class="flex items-center justify-between">
                  <p class="truncate text-xs font-medium text-gray-900">{{ task.file.name }}</p>
                  <span class="text-[10px] text-gray-400">{{ formatFileSize(task.file.size) }}</span>
                </div>

                <!-- 单个进度条 -->
                <div v-if="task.status === 'uploading'" class="mt-1">
                  <div class="h-1 w-full overflow-hidden rounded-full bg-gray-100">
                    <div
                        class="h-full bg-blue-500 transition-all duration-200"
                        :style="{ width: `${task.progress}%` }"
                    />
                  </div>
                </div>
                <p v-else-if="task.error" class="mt-0.5 text-[10px] text-red-500 truncate">{{ task.error }}</p>
                <p v-else-if="task.status === 'success'" class="mt-0.5 text-[10px] text-green-600">上传完成</p>
                <p v-else class="mt-0.5 text-[10px] text-gray-400">等待中...</p>
              </div>

              <!-- 按钮 -->
              <div class="flex items-center opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                    v-if="task.status === 'error'"
                    @click.stop="retryUpload(task)"
                    class="rounded-lg p-1 text-blue-600 hover:bg-blue-50 transition-colors"
                    title="重试"
                >
                  <ArrowPathIcon class="h-3.5 w-3.5"/>
                </button>
                <button
                    @click.stop="removeTask(task.id)"
                    class="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-red-500 transition-colors"
                    title="移除"
                >
                  <XMarkIcon class="h-3.5 w-3.5"/>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import {watch} from 'vue'
import {useUIStore} from '@/stores/ui'
import {useImageStore} from '@/stores/image'
import {imageApi} from '@/api/image'
import {createThumbnail} from '@/utils/image'
import type {UploadTask} from '@/stores/ui'
import {
  ArrowUpTrayIcon,
  PhotoIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  ArrowPathIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'

interface Props {
  collapsed?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  collapsed: true
})

const uiStore = useUIStore()
const imageStore = useImageStore()

function toggleDrawer() {
  if (uiStore.uploadDrawerOpen) {
    uiStore.closeUploadDrawer()
  } else {
    uiStore.openUploadDrawer()
  }
}

// 监听上传任务，自动上传
watch(() => uiStore.uploadTasks, (tasks) => {
  tasks.forEach(task => {
    if (task.status === 'pending') {
      uploadFile(task)
    }
  })
}, {deep: true})

async function uploadFile(task: UploadTask) {
  try {
    // 生成预览图 (使用缩略图以节省内存)
    uiStore.updateUploadTask(task.id, {
      status: 'uploading',
    })

    // 异步生成缩略图，不阻塞上传开始
    createThumbnail(task.file).then(imageUrl => {
      if (imageUrl) {
        uiStore.updateUploadTask(task.id, {imageUrl})
      }
    }).catch(console.error)

    // 上传文件
    await imageApi.upload(task.file, (progress) => {
      uiStore.updateUploadTask(task.id, {progress})
    })

    // 上传成功
    uiStore.updateUploadTask(task.id, {
      status: 'success',
      progress: 100,
    })

    // 刷新图片列表
    await imageStore.refreshImages()
  } catch (error) {
    // 上传失败
    uiStore.updateUploadTask(task.id, {
      status: 'error',
      error: error instanceof Error ? error.message : '上传失败',
    })
  }
}

async function retryUpload(task: UploadTask) {
  uiStore.updateUploadTask(task.id, {
    status: 'pending',
    progress: 0,
    error: undefined,
  })
}

function removeTask(taskId: string) {
  uiStore.removeUploadTask(taskId)
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>