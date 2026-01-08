<template>
  <div
      :class="{ 'aspect-square': square }"
      class="group relative cursor-pointer overflow-hidden rounded-xl bg-white/5 transition-all duration-300 hover:scale-[1.02]"
      @click="$emit('click')"
      @contextmenu.prevent="$emit('contextmenu', $event)"
  >
    <!-- 图片 -->
    <CachedImage
        v-if="image&&imageUrl"
        :alt="image.original_name"
        :draggable="false"
        :img-class="['w-full object-cover transition-transform duration-500', { 'h-full': square }]"
        :src="imageUrl"
        @error="handleImageError"
    />

    <!-- 加载占位符 -->
    <div v-else class="flex items-center justify-center bg-white/5" :class="[square ? 'h-full' : 'min-h-[200px]']">
      <PhotoIcon class="h-10 w-10 text-gray-700"/>
    </div>

    <!-- 极简内描边 (替代边框，更有质感) -->
    <div
        class="absolute inset-0 rounded-2xl ring-1 ring-inset ring-white/10 pointer-events-none transition-opacity group-hover:ring-white/20"></div>

    <!-- 悬停信息 -->
    <div
        v-if="image"
        class="absolute bottom-0 left-0 right-0 translate-y-full bg-linear-to-t from-black/90 via-black/50 to-transparent p-4 transition-transform duration-300 ease-out group-hover:translate-y-0">
      <p class="truncate text-sm font-medium text-gray-100">{{ image.original_name }}</p>
      <div class="mt-1 flex items-center gap-2 text-xs text-gray-400 font-light">
        <span v-if="image.taken_at">{{ formatDate(image.taken_at) }}</span>
        <span v-if="image.camera_model" class="truncate opacity-80">{{ image.camera_model }}</span>
        <span v-if="image.location_name" class="truncate">{{ image.location_name }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {PhotoIcon} from '@heroicons/vue/24/outline'
import CachedImage from '@/components/widgets/common/CachedImage.vue'
import type {Image} from '@/types'

interface Props {
  image: Image | null | undefined
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
  if (!props.image) return null

  // 优先使用缩略图 URL
  if (props.image.thumbnail_url) {
    return props.image.thumbnail_url
  }

  // 否则使用原图 URL
  return props.image.url
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
