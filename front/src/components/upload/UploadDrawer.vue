<template>
  <!-- 收起状态的进度条 (嵌入在 Sidebar 中) -->
  <div
      v-if="uiStore.uploadTasks.length > 0"
      class="mb-2"
  >
    <div
        class="flex cursor-pointer items-center gap-3 rounded-xl transition-all duration-300 hover:bg-white/5 hover:scale-105 active:scale-95"
        :class="[
        props.collapsed ? 'justify-center border-0 p-1' : 'border border-white/10 bg-white/5 p-2 shadow-lg px-3 backdrop-blur-sm'
      ]"
        @click="toggleDrawer"
    >
      <!-- 环形进度条 -->
      <div class="relative flex h-9 w-9 flex-shrink-0 items-center justify-center">
        <svg class="h-full w-full -rotate-90 transform" viewBox="0 0 36 36">
          <!-- 背景圆 -->
          <path
              class="text-white/10"
              d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
          />
          <!-- 进度圆 - Neon Glow -->
          <path
              class="text-primary-500 transition-all duration-500 ease-[cubic-bezier(0.16,1,0.3,1)] drop-shadow-[0_0_4px_rgba(139,92,246,0.6)]"
              :stroke-dasharray="`${uiStore.totalProgress}, 100`"
              d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
              stroke-linecap="round"
          />
        </svg>
        <ArrowUpTrayIcon class="absolute h-4 w-4 text-primary-400 animate-pulse"/>
      </div>

      <div v-if="!props.collapsed" class="mr-1 flex flex-1 flex-col min-w-0">
        <span class="text-sm font-bold text-white truncate tracking-wide">
          {{ uiStore.uploadingCount > 0 ? '正在上传...' : '上传完成' }}
        </span>
        <span class="text-xs font-medium text-gray-400 truncate">
          <span v-if="uiStore.failedCount>0">
               ( {{ uiStore.completedCount }} + <span class="text-red-400">{{
              uiStore.failedCount
            }}</span> )
          </span>
          <span v-else>
          {{ uiStore.completedCount }}
          </span>
          <span class="text-gray-600">/</span> {{ uiStore.uploadTasks.length }}
        </span>
      </div>
    </div>
  </div>

  <!-- 展开状态的详情框 (左下角 Popover，定位在 Sidebar 旁) -->
  <Teleport to="body">
    <Transition
        enter-active-class="transition duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
        enter-from-class="transform scale-95 opacity-0 translate-y-4 -translate-x-4 blur-sm"
        enter-to-class="transform scale-100 opacity-100 translate-y-0 translate-x-0 blur-0"
        leave-active-class="transition duration-200 ease-in"
        leave-from-class="transform scale-100 opacity-100 translate-y-0 translate-x-0 blur-0"
        leave-to-class="transform scale-95 opacity-0 translate-y-4 -translate-x-4 blur-sm"
    >
      <div
          v-if="uiStore.uploadDrawerOpen"
          class="fixed z-50 flex max-h-[600px] w-80 flex-col overflow-hidden rounded-2xl border border-white/10 bg-black/80 shadow-[0_20px_50px_-12px_rgba(0,0,0,0.5)] backdrop-blur-xl origin-bottom-left"
          :class="[props.collapsed ? 'left-20 bottom-4' : 'left-68 bottom-4']"
          :style="{ left: props.collapsed ? '5.5rem' : '17.5rem' }"
      >
        <!-- 头部 -->
        <div class="flex items-center justify-between border-b border-white/5 bg-white/5 p-4 backdrop-blur-md">
          <div class="flex items-center gap-3">
            <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-500/20 text-primary-400 shadow-[0_0_10px_rgba(139,92,246,0.2)]">
              <ArrowUpTrayIcon class="h-4 w-4"/>
            </div>
            <div>
              <h3 class="text-sm font-bold text-white tracking-wide">上传队列</h3>
              <p class="text-xs text-gray-400">{{ uiStore.uploadTasks.length }} 个任务</p>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <button
                v-if="uiStore.completedCount > 0 || uiStore.failedCount > 0"
                @click="uiStore.clearCompletedTasks"
                class="rounded-lg px-2.5 py-1 text-xs font-medium text-gray-400 hover:bg-white/10 hover:text-white transition-colors"
            >
              清除
            </button>
            <button
                @click="uiStore.closeUploadDrawer"
                class="rounded-full p-1.5 text-gray-400 hover:bg-white/10 hover:text-white transition-colors"
            >
              <XMarkIcon class="h-4 w-4"/>
            </button>
          </div>
        </div>

        <!-- 总体进度 -->
        <div v-if="uiStore.uploadingCount > 0" class="border-b border-white/5 bg-black/20 px-4 py-3">
          <div class="mb-2 flex items-center justify-between text-xs">
            <span class="font-medium text-gray-400">总体进度</span>
            <span class="font-bold text-primary-400">{{ uiStore.totalProgress }}%</span>
          </div>
          <div class="h-1.5 w-full overflow-hidden rounded-full bg-white/10">
            <div
                class="h-full bg-gradient-to-r from-primary-600 to-primary-400 transition-all duration-300 ease-out shadow-[0_0_10px_rgba(139,92,246,0.5)]"
                :style="{ width: `${uiStore.totalProgress}%` }"
            />
          </div>
        </div>

        <!-- 任务列表 -->
        <RecycleScroller
            v-slot="{ item: task }"
            :item-size="56"
            :items="uiStore.uploadTasks"
            class="flex-1 overflow-y-auto p-2 scrollbar-thin scrollbar-thumb-white/10 hover:scrollbar-thumb-white/20"
            key-field="id"
        >
          <div class="space-y-1">
            <div
                class="group flex items-center gap-3 rounded-xl p-2 transition-all hover:bg-white/5 border border-transparent hover:border-white/5"
            >
              <!-- 缩略图 -->
              <div
                  class="relative h-10 w-10 flex-shrink-0 overflow-hidden rounded-lg bg-white/5 ring-1 ring-white/10">
                <img
                    v-if="task.imageUrl"
                    :src="task.imageUrl"
                    :alt="task.file.name"
                    class="h-full w-full object-cover"
                />
                <div v-else class="flex h-full items-center justify-center">
                  <PhotoIcon class="h-5 w-5 text-gray-600"/>
                </div>

                <!-- 状态覆盖层 -->
                <div
                    v-if="task.status !== 'pending'"
                    class="absolute inset-0 flex items-center justify-center bg-black/40 transition-opacity"
                    :class="{'bg-red-500/20': task.status === 'error'}"
                >
                  <CheckCircleIcon v-if="task.status === 'success'" class="h-5 w-5 text-green-400 drop-shadow-[0_0_5px_rgba(74,222,128,0.5)]"/>
                  <ExclamationCircleIcon v-else-if="task.status === 'error'"
                                         class="h-5 w-5 text-red-400 drop-shadow-[0_0_5px_rgba(248,113,113,0.5)]"/>
                  <div v-else-if="task.status === 'uploading'"
                       class="h-4 w-4 rounded-full border-2 border-primary-400 border-t-transparent animate-spin"/>
                </div>
              </div>

              <!-- 信息 -->
              <div class="flex-1 min-w-0 py-0.5">
                <div class="flex items-center justify-between">
                  <p class="truncate text-xs font-medium text-gray-200">{{ task.file.name }}</p>
                  <span class="text-[10px] text-gray-500">{{ formatFileSize(task.file.size) }}</span>
                </div>

                <!-- 单个进度条 -->
                <div v-if="task.status === 'uploading'" class="mt-1.5">
                  <div class="h-1 w-full overflow-hidden rounded-full bg-white/10">
                    <div
                        class="h-full bg-primary-500 transition-all duration-200 shadow-[0_0_5px_rgba(139,92,246,0.4)]"
                        :style="{ width: `${task.progress}%` }"
                    />
                  </div>
                </div>
                <p v-else-if="task.error" class="mt-0.5 text-[10px] text-red-400 truncate">{{ task.error }}</p>
                <p v-else-if="task.status === 'success'" class="mt-0.5 text-[10px] text-green-400">上传完成</p>
                <p v-else class="mt-0.5 text-[10px] text-gray-500">等待中...</p>
              </div>

              <!-- 按钮 -->
              <div class="flex items-center opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                    v-if="task.status === 'error'"
                    @click.stop="retryUpload(task)"
                    class="rounded-lg p-1.5 text-primary-400 hover:bg-primary-500/10 transition-colors"
                    title="重试"
                >
                  <ArrowPathIcon class="h-3.5 w-3.5"/>
                </button>
                <button
                    @click.stop="removeTask(task.id)"
                    class="rounded-lg p-1.5 text-gray-500 hover:bg-white/10 hover:text-red-400 transition-colors"
                    title="移除"
                >
                  <XMarkIcon class="h-3.5 w-3.5"/>
                </button>
              </div>
            </div>
          </div>
        </RecycleScroller>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import type {UploadTask} from '@/stores/ui'
import {useUIStore} from '@/stores/ui'
import {RecycleScroller} from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'

import {
  ArrowPathIcon,
  ArrowUpTrayIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  PhotoIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'

interface Props {
  collapsed?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  collapsed: true
})

const uiStore = useUIStore()

function toggleDrawer() {
  if (uiStore.uploadDrawerOpen) {
    uiStore.closeUploadDrawer()
  } else {
    uiStore.openUploadDrawer()
  }
}

async function retryUpload(task: UploadTask) {
  uiStore.retryUploadTask(task.id)
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