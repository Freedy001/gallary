<template>
  <AppLayout>
    <template #default>
      <!-- 命令面板 -->
      <CommandPalette />

      <!-- 图片网格 -->
      <ImageGrid/>
    </template>

    <template #overlay>
      <!-- 悬浮时间线 -->
      <Timeline />
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useImageStore } from '@/stores/image'
import AppLayout from '@/components/layout/AppLayout.vue'
import CommandPalette from '@/components/search/CommandPalette.vue'
import ImageGrid from '@/components/gallery/ImageGrid.vue'
import Timeline from '@/components/gallery/Timeline.vue'

const imageStore = useImageStore()

onMounted(async () => {
  // 加载图片列表
  if (imageStore.images.length === 0) {
    await imageStore.fetchImages()
  }
})
</script>
