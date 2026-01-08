<template>
  <img
    v-if="cachedUrl"
    :src="cachedUrl"
    :alt="alt"
    :class="imgClass"
    :loading="loading"
    :draggable="draggable"
    @error="$emit('error', $event)"
    @load="$emit('load', $event)"
  />
  <slot v-else name="placeholder">
    <div :class="placeholderClass"></div>
  </slot>
</template>

<script setup lang="ts">
import {onMounted, ref, watch} from 'vue'
import {thumbnailCache} from '@/utils/imageCache.ts'

const props = withDefaults(defineProps<{
  src: string
  alt?: string
  imgClass?: string | string[] | Record<string, boolean> | (string | Record<string, boolean>)[]
  placeholderClass?: string
  loading?: 'lazy' | 'eager'
  draggable?: boolean
  preload?: boolean // 是否立即预加载到缓存
}>(), {
  alt: '',
  imgClass: '',
  placeholderClass: '',
  loading: 'lazy',
  draggable: false,
  preload: true
})

defineEmits<{
  error: [event: Event]
  load: [event: Event]
}>()

const cachedUrl = ref<string>('')

async function loadImage() {
  if (!props.src) {
    cachedUrl.value = ''
    return
  }

  // 先检查缓存
  if (thumbnailCache.has(props.src)) {
    cachedUrl.value = thumbnailCache.get(props.src)
    return
  }

  // 如果需要预加载，先加载到缓存
  if (props.preload) {
    try {
      cachedUrl.value = await thumbnailCache.preload(props.src)
    } catch {
      // 加载失败时使用原始 URL
      cachedUrl.value = props.src
    }
  } else {
    // 不预加载，直接使用原始 URL（浏览器会自动缓存）
    cachedUrl.value = props.src
  }
}

watch(() => props.src, loadImage, { immediate: true })

onMounted(() => {
  if (!cachedUrl.value && props.src) {
    loadImage()
  }
})
</script>
