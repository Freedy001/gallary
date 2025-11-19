<template>
  <Teleport to="body">
    <!-- 收起状态的进度条 -->
    <Transition name="slide-up">
      <div
        v-if="uiStore.uploadTasks.length > 0 && !uiStore.uploadDrawerOpen"
        class="fixed bottom-0 left-0 right-0 z-40 cursor-pointer border-t border-gray-200 bg-white p-4 shadow-lg"
        @click="uiStore.openUploadDrawer"
      >
        <div class="mx-auto max-w-4xl">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-full bg-blue-100">
                <ArrowUpTrayIcon class="h-5 w-5 text-blue-600" />
              </div>
              <div>
                <p class="text-sm font-medium text-gray-900">
                  正在上传 {{ uiStore.uploadingCount }} 张图片
                </p>
                <p class="text-xs text-gray-500">
                  {{ uiStore.completedCount }} / {{ uiStore.uploadTasks.length }} 已完成
                </p>
              </div>
            </div>
            <div class="flex items-center gap-3">
              <div class="h-2 w-48 overflow-hidden rounded-full bg-gray-200">
                <div
                  class="h-full bg-blue-600 transition-all duration-300"
                  :style="{ width: `${uiStore.totalProgress}%` }"
                />
              </div>
              <span class="text-sm font-medium text-gray-700">{{ uiStore.totalProgress }}%</span>
              <ChevronUpIcon class="h-5 w-5 text-gray-400" />
            </div>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 展开状态的抽屉 -->
    <Transition name="slide-up">
      <div
        v-if="uiStore.uploadDrawerOpen"
        class="fixed bottom-0 left-0 right-0 z-40 flex max-h-[70vh] flex-col border-t border-gray-200 bg-white shadow-2xl"
      >
        <!-- 抽屉头部 -->
        <div class="flex items-center justify-between border-b border-gray-200 p-4">
          <div class="flex items-center gap-3">
            <h3 class="text-lg font-semibold text-gray-900">上传管理</h3>
            <span class="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700">
              {{ uiStore.uploadTasks.length }} 个任务
            </span>
          </div>

          <div class="flex items-center gap-2">
            <button
              v-if="uiStore.completedCount > 0 || uiStore.failedCount > 0"
              @click="uiStore.clearCompletedTasks"
              class="rounded-lg px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100"
            >
              清除已完成
            </button>
            <button
              @click="uiStore.closeUploadDrawer"
              class="rounded-lg p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
            >
              <ChevronDownIcon class="h-5 w-5" />
            </button>
          </div>
        </div>

        <!-- 总体进度 -->
        <div class="border-b border-gray-200 p-4">
          <div class="mb-2 flex items-center justify-between text-sm">
            <span class="text-gray-700">
              {{ uiStore.completedCount }} / {{ uiStore.uploadTasks.length }} 已完成
            </span>
            <span class="font-medium text-gray-900">{{ uiStore.totalProgress }}%</span>
          </div>
          <div class="h-2 w-full overflow-hidden rounded-full bg-gray-200">
            <div
              class="h-full bg-blue-600 transition-all duration-300"
              :style="{ width: `${uiStore.totalProgress}%` }"
            />
          </div>
        </div>

        <!-- 上传任务列表 -->
        <div class="flex-1 overflow-y-auto p-4">
          <div class="space-y-3">
            <div
              v-for="task in uiStore.uploadTasks"
              :key="task.id"
              class="flex items-center gap-3 rounded-lg border border-gray-200 p-3"
            >
              <!-- 缩略图预览 -->
              <div class="relative h-16 w-16 flex-shrink-0 overflow-hidden rounded-lg bg-gray-100">
                <img
                  v-if="task.imageUrl"
                  :src="task.imageUrl"
                  :alt="task.file.name"
                  class="h-full w-full object-cover"
                />
                <div v-else class="flex h-full items-center justify-center">
                  <PhotoIcon class="h-8 w-8 text-gray-400" />
                </div>

                <!-- 状态图标 -->
                <div class="absolute inset-0 flex items-center justify-center bg-black bg-opacity-50">
                  <CheckCircleIcon v-if="task.status === 'success'" class="h-6 w-6 text-green-400" />
                  <ExclamationCircleIcon v-else-if="task.status === 'error'" class="h-6 w-6 text-red-400" />
                  <ArrowPathIcon v-else-if="task.status === 'uploading'" class="h-6 w-6 animate-spin text-white" />
                </div>
              </div>

              <!-- 文件信息 -->
              <div class="flex-1 min-w-0">
                <p class="truncate text-sm font-medium text-gray-900">{{ task.file.name }}</p>
                <p class="text-xs text-gray-500">{{ formatFileSize(task.file.size) }}</p>

                <!-- 进度条 -->
                <div v-if="task.status === 'uploading'" class="mt-2">
                  <div class="h-1 w-full overflow-hidden rounded-full bg-gray-200">
                    <div
                      class="h-full bg-blue-600 transition-all duration-200"
                      :style="{ width: `${task.progress}%` }"
                    />
                  </div>
                </div>

                <!-- 错误信息 -->
                <p v-if="task.error" class="mt-1 text-xs text-red-600">{{ task.error }}</p>
              </div>

              <!-- 操作按钮 -->
              <div class="flex items-center gap-2">
                <button
                  v-if="task.status === 'error'"
                  @click="retryUpload(task)"
                  class="rounded-lg p-2 text-blue-600 hover:bg-blue-50"
                  title="重试"
                >
                  <ArrowPathIcon class="h-4 w-4" />
                </button>
                <button
                  @click="removeTask(task.id)"
                  class="rounded-lg p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                  title="移除"
                >
                  <XMarkIcon class="h-4 w-4" />
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
import { watch } from 'vue'
import { useUIStore } from '@/stores/ui'
import { useImageStore } from '@/stores/image'
import { imageApi } from '@/api/image'
import type { UploadTask } from '@/stores/ui'
import {
  ArrowUpTrayIcon,
  ChevronUpIcon,
  ChevronDownIcon,
  PhotoIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  ArrowPathIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'

const uiStore = useUIStore()
const imageStore = useImageStore()

// 监听上传任务，自动上传
watch(() => uiStore.uploadTasks, (tasks) => {
  tasks.forEach(task => {
    if (task.status === 'pending') {
      uploadFile(task)
    }
  })
}, { deep: true })

async function uploadFile(task: UploadTask) {
  try {
    // 生成预览图
    const imageUrl = URL.createObjectURL(task.file)
    uiStore.updateUploadTask(task.id, {
      status: 'uploading',
      imageUrl,
    })

    // 上传文件
    const response = await imageApi.upload(task.file, (progress) => {
      uiStore.updateUploadTask(task.id, { progress })
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
