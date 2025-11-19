<template>
  <div class="p-6">
    <!-- 加载状态 -->
    <div v-if="imageStore.loading && (!imageStore.images || imageStore.images.length === 0)" class="py-12 text-center">
      <div class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-blue-600"></div>
      <p class="mt-4 text-sm text-gray-600">加载中...</p>
    </div>

    <!-- 空状态 -->
    <div v-else-if="!imageStore.images || imageStore.images.length === 0" class="flex min-h-[60vh] items-center justify-center py-20">
      <div class="text-center">
        <div class="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-gray-100">
          <PhotoIcon class="h-12 w-12 text-gray-400" />
        </div>
        <h3 class="mt-6 text-xl font-semibold text-gray-900">还没有图片</h3>
        <p class="mt-3 text-base text-gray-600">上传您的第一张图片开始使用</p>
      </div>
    </div>

    <!-- 图片网格 -->
    <div v-else>
      <div
        :class="[
          'grid gap-2',
          gridClass,
        ]"
      >
        <ImageCard
          v-for="(image, index) in imageStore.images"
          :key="image.id"
          :image="image"
          :index="index"
          @click="handleImageClick(index)"
        />
      </div>

      <!-- 加载更多 -->
      <div
        v-if="imageStore.hasMore"
        ref="loadMoreRef"
        class="mt-8 flex justify-center"
      >
        <button
          @click="loadMore"
          :disabled="imageStore.loading"
          class="rounded-lg border border-gray-300 px-6 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
        >
          {{ imageStore.loading ? '加载中...' : '加载更多' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useImageStore } from '@/stores/image'
import { useUIStore } from '@/stores/ui'
import { PhotoIcon } from '@heroicons/vue/24/outline'
import ImageCard from './ImageCard.vue'

const imageStore = useImageStore()
const uiStore = useUIStore()
const loadMoreRef = ref<HTMLElement>()

// 根据密度动态计算网格列数
const gridClass = computed(() => {
  const columns = uiStore.gridColumns
  const desktopClass = {
    1: 'md:grid-cols-1',
    2: 'md:grid-cols-2',
    3: 'md:grid-cols-3',
    4: 'md:grid-cols-4',
    5: 'md:grid-cols-5',
    6: 'md:grid-cols-6',
  }[columns.desktop] || 'md:grid-cols-4'

  const tabletClass = {
    1: 'sm:grid-cols-1',
    2: 'sm:grid-cols-2',
    3: 'sm:grid-cols-3',
    4: 'sm:grid-cols-4',
  }[columns.tablet] || 'sm:grid-cols-2'

  const mobileClass = columns.mobile === 1 ? 'grid-cols-1' : 'grid-cols-2'

  return `${mobileClass} ${tabletClass} ${desktopClass}`
})

function handleImageClick(index: number) {
  uiStore.openImageViewer(index)
}

async function loadMore() {
  if (!imageStore.loading && imageStore.hasMore) {
    await imageStore.loadMore()
  }
}

onMounted(() => {
  // 实现无限滚动
  const observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting && !imageStore.loading && imageStore.hasMore) {
        loadMore()
      }
    },
    { threshold: 0.1 }
  )

  if (loadMoreRef.value) {
    observer.observe(loadMoreRef.value)
  }

  return () => {
    observer.disconnect()
  }
})
</script>
