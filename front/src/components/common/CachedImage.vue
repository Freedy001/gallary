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
import {onBeforeUnmount, onMounted, ref, watch} from 'vue'
import {thumbnailCache} from '@/utils/imageCache.ts'

const props = withDefaults(defineProps<{
  src: string
  alt?: string
  imgClass?: string | string[] | Record<string, boolean> | (string | Record<string, boolean>)[]
  placeholderClass?: string
  loading?: 'lazy' | 'eager'
  draggable?: boolean
  preload?: boolean // 是否立即预加载到缓存
  loadPriority?: number // 加载优先级，数值越小优先级越高
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
let currentSrc = '' // 跟踪当前正在使用的 src，用于引用计数

async function loadImage() {
  const newSrc = props.src

  // 释放旧的引用
  if (currentSrc && currentSrc !== newSrc) {
    thumbnailCache.release(currentSrc)
  }

  if (!newSrc) {
    cachedUrl.value = ''
    currentSrc = ''
    return
  }

  currentSrc = newSrc

  // 先检查缓存
  if (thumbnailCache.has(newSrc)) {
    cachedUrl.value = thumbnailCache.get(newSrc)
    thumbnailCache.retain(newSrc)
    return
  }

  // 如果需要预加载，先加载到缓存
  if (props.preload) {
    try {
      cachedUrl.value = await thumbnailCache.preload(newSrc, props.loadPriority)
      thumbnailCache.retain(newSrc)
    } catch {
      // 加载失败时使用原始 URL
      cachedUrl.value = newSrc
    }
  } else {
    // 不预加载，直接使用原始 URL（浏览器会自动缓存）
    cachedUrl.value = newSrc
  }
}

watch(() => props.src, loadImage, { immediate: true })

onMounted(() => {
  if (!cachedUrl.value && props.src) {
    loadImage()
  }
})

onBeforeUnmount(() => {
  // 组件卸载时释放引用
  if (currentSrc) {
    thumbnailCache.release(currentSrc)
  }
})
</script>
