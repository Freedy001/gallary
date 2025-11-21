<template>
  <div
    class="group relative cursor-pointer overflow-hidden rounded-lg border border-gray-200 bg-gray-100 transition-all hover:shadow-md"
    :class="{ 'aspect-square': square }"
    @click="$emit('click')"
    @contextmenu.prevent="$emit('contextmenu', $event)"
  >
    <!-- 图片 -->
    <img
      v-if="imageUrl"
      :src="imageUrl"
      :alt="image.original_name"
      class="w-full object-cover transition-transform duration-300 group-hover:scale-105"
      :class="{ 'h-full': square }"
      loading="lazy"
      @error="handleImageError"
    />

    <!-- 加载占位符 -->
    <div v-else class="flex items-center justify-center" :class="[square ? 'h-full' : 'min-h-[200px]']">
      <PhotoIcon class="h-12 w-12 text-gray-400" />
    </div>

    <!-- 悬停遮罩 -->
    <div class="absolute inset-0 bg-black/0 transition-colors group-hover:bg-black/20" />

    <!-- 悬停信息 -->
    <div class="absolute bottom-0 left-0 right-0 translate-y-full bg-gradient-to-t from-black/70 to-transparent p-3 transition-transform group-hover:translate-y-0">
      <p class="truncate text-sm font-medium text-white">{{ image.original_name }}</p>
      <div class="mt-1 flex items-center gap-2 text-xs text-gray-300">
        <span v-if="image.taken_at">{{ formatDate(image.taken_at) }}</span>
        <span v-if="image.camera_model">{{ image.camera_model }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { imageApi } from '@/api/image'
import { PhotoIcon } from '@heroicons/vue/24/outline'
import type { Image } from '@/types'

interface Props {
  image: Image
  index: number
  square?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  square: true
})
defineEmits<{
  click: []
  contextmenu: [event: MouseEvent]
}>()

const imageError = ref(false)

const imageUrl = computed(() => {
  if (imageError.value) return null

  // 优先使用缩略图
  if (props.image.thumbnail_path) {
    return imageApi.getImageUrl(props.image.thumbnail_path)
  }

  // 否则使用原图
  return imageApi.getImageUrl(props.image.storage_path)
})

function handleImageError() {
  imageError.value = true
}

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
}
</script>
