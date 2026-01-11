<template>
  <div
      :class="[
        selected
          ? 'ring-2 ring-primary-500 shadow-[0_0_20px_rgba(139,92,246,0.3)]'
          : album.is_smart_album
            ? 'ring-purple-500/30 hover:ring-primary-500/50'
            : 'ring-white/10 hover:ring-primary-500/50'
      ]"
      class="relative aspect-square rounded-2xl overflow-hidden bg-white/5 ring-1 transition-all duration-150 hover:shadow-[0_0_30px_rgba(139,92,246,0.2)] hover:scale-[1.01]"
  >
    <!-- 内容容器 - 禁用指针事件以让拖拽穿透 -->
    <div class="absolute inset-0 pointer-events-none">
      <!-- 选中勾选标记 -->
      <div
          v-if="selected"
          class="absolute top-3 left-3 z-10 flex h-6 w-6 items-center justify-center rounded-full bg-primary-500 text-white shadow-lg"
      >
        <CheckIcon class="h-4 w-4" />
      </div>

      <!-- 封面图 -->
      <img
          v-if="album.cover_image?.thumbnail_url"
          :alt="album.name"
          :src="album.cover_image.thumbnail_url"
          class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
      />
      <div v-else
           class="w-full h-full flex items-center justify-center bg-linear-to-br from-gray-800/50 to-gray-900/50">
        <RectangleStackIcon class="h-12 w-12 text-white/20"/>
      </div>

      <!-- 渐变遮罩 -->
      <div class="absolute inset-0 bg-linear-to-t from-black/80 via-black/20 to-transparent"></div>

      <!-- 智能相册标识 -->
      <div v-if="album.is_smart_album && !selected" class="absolute top-3 left-3">
        <div class="flex items-center gap-1 px-2 py-1 rounded-lg bg-purple-500/20 backdrop-blur-sm border border-purple-500/30">
          <SparklesIcon class="h-3 w-3 text-purple-400"/>
          <span class="text-[10px] text-purple-300 font-medium">AI</span>
        </div>
      </div>

      <!-- 相册信息 -->
      <div class="absolute bottom-0 left-0 right-0 p-4">
        <h3 class="text-white font-medium truncate">{{ album.name }}</h3>
        <div class="flex items-center justify-between mt-1">
          <p class="text-xs text-gray-400">{{ album.image_count }} 张照片</p>
          <!-- 显示关联度 -->
          <div v-if="showProbability && album.hdbscan_avg_probability"
               :title="`照片内容相似度: ${(album.hdbscan_avg_probability * 100).toFixed(0)}%`"
               class="flex items-center gap-1.5 bg-black/20 px-2 py-0.5 rounded-full backdrop-blur-xs">
            <div :class="['w-1.5 h-1.5 rounded-full shadow-[0_0_5px_currentColor]', probabilityColor]"></div>
            <span class="text-[10px] text-gray-300 font-medium tracking-tight">相似度 {{ (album.hdbscan_avg_probability * 100).toFixed(0) }}%</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 菜单按钮 - 保留指针事件 -->
    <button
        class="absolute top-3 right-3 p-2 rounded-lg bg-black/40 text-white/60 opacity-0 group-hover:opacity-100 hover:bg-white/20 hover:text-white transition-all z-20"
        @click.stop="emit('menu', $event)"
    >
      <EllipsisHorizontalIcon class="h-5 w-5"/>
    </button>
  </div>
</template>

<script lang="ts" setup>
import {computed} from 'vue'
import {CheckIcon, EllipsisHorizontalIcon, RectangleStackIcon, SparklesIcon} from '@heroicons/vue/24/outline'
import type {Album} from '@/types'

const props = defineProps<{
  album: Album
  showProbability?: boolean
  selected?: boolean
}>()

const emit = defineEmits<{
  menu: [event: MouseEvent]
}>()

// 根据概率值计算颜色
const probabilityColor = computed(() => {
  const prob = props.album?.hdbscan_avg_probability ?? 0
  if (prob >= 0.8) return 'bg-emerald-400 text-emerald-400'
  if (prob >= 0.6) return 'bg-blue-400 text-blue-400'
  if (prob >= 0.4) return 'bg-yellow-400 text-yellow-400'
  return 'bg-rose-400 text-rose-400'
})
</script>
