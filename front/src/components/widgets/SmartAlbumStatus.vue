<template>
  <!-- 展开状态 -->
  <div
    v-if="!collapsed"
    class="group cursor-pointer -mx-2 px-2 py-2 rounded-lg bg-white/5 border border-white/5 hover:bg-white/10 transition-colors"
    @click="handleClick"
  >
    <!-- 进行中 -->
    <template v-if="smartAlbumStore.taskInProgress">
      <div class="flex items-center justify-between text-xs text-gray-400 mb-1.5">
        <div class="flex items-center gap-1.5 font-mono tracking-wider">
          <SparklesIcon class="h-3.5 w-3.5 text-primary-400" />
          <span>智能相册</span>
        </div>
        <span class="text-xs text-primary-300">{{ smartAlbumStore.currentProgress?.progress || 0 }}%</span>
      </div>

      <div class="space-y-1.5">
        <div class="text-xs text-gray-300 truncate">
          {{ smartAlbumStore.currentProgress?.message || '处理中...' }}
        </div>
        <!-- 进度条 -->
        <div class="h-1 bg-white/5 rounded-full overflow-hidden">
          <div
            :style="{ width: `${smartAlbumStore.currentProgress?.progress || 0}%` }"
            class="h-full rounded-full transition-all duration-300 bg-gradient-to-r from-purple-500 to-blue-500"
          ></div>
        </div>
      </div>
    </template>

    <!-- 已完成 -->
    <template v-else-if="smartAlbumStore.result">
      <div class="flex items-center justify-between text-xs text-green-400 mb-1.5">
        <div class="flex items-center gap-1.5 font-mono tracking-wider">
          <CheckCircleIcon class="h-3.5 w-3.5" />
          <span>生成完成</span>
        </div>
        <ArrowRightIcon class="h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>
      <div class="text-xs text-gray-400">
        已创建 {{ smartAlbumStore.result.cluster_count }} 个相册
      </div>
    </template>

    <!-- 失败 -->
    <template v-else-if="smartAlbumStore.errorMessage">
      <div class="flex items-center justify-between text-xs text-red-400 mb-1.5">
        <div class="flex items-center gap-1.5 font-mono tracking-wider">
          <XCircleIcon class="h-3.5 w-3.5" />
          <span>生成失败</span>
        </div>
        <ArrowRightIcon class="h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>
      <div class="text-xs text-gray-400 truncate">
        {{ smartAlbumStore.errorMessage }}
      </div>
    </template>
  </div>

  <!-- 收起状态 -->
  <div
    v-else
    class="group cursor-pointer -mx-2 px-2 py-2 rounded-lg flex flex-col items-center gap-1 hover:bg-white/10 transition-colors"
    @click="handleClick"
  >
    <div class="relative">
      <SparklesIcon
        :class="{
          'text-primary-400': smartAlbumStore.taskInProgress,
          'text-green-400': smartAlbumStore.result,
          'text-red-400': smartAlbumStore.errorMessage
        }"
        class="h-4 w-4"
      />
      <span
        v-if="smartAlbumStore.taskInProgress"
        class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-primary-500 animate-pulse border border-gray-900"
      ></span>
      <span
        v-else-if="smartAlbumStore.result"
        class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-green-500 border border-gray-900"
      ></span>
      <span
        v-else-if="smartAlbumStore.errorMessage"
        class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-red-500 border border-gray-900"
      ></span>
    </div>
    <span :class="{
        'text-primary-400': smartAlbumStore.taskInProgress,
        'text-green-400': smartAlbumStore.result,
        'text-red-400': smartAlbumStore.errorMessage
      }"
      class="text-[10px] font-bold"
    >
      {{ smartAlbumStore.taskInProgress ? `${smartAlbumStore.currentProgress?.progress || 0}%` : (smartAlbumStore.result ? '完成' : '失败') }}
    </span>
  </div>
</template>

<script lang="ts" setup>
import {useSmartAlbumStore} from '@/stores/smartAlbum'
import {ArrowRightIcon, CheckCircleIcon, SparklesIcon, XCircleIcon,} from '@heroicons/vue/24/outline'

defineProps<{
  collapsed: boolean
}>()

const emit = defineEmits<{
  click: []
}>()

const smartAlbumStore = useSmartAlbumStore()

function handleClick() {
  // 如果任务已完成或失败，触发点击事件
  if (smartAlbumStore.result || smartAlbumStore.errorMessage) {
    emit('click')
  }
}
</script>
