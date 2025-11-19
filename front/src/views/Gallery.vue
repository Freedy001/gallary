<template>
  <AppLayout>
    <!-- 命令面板 -->
    <CommandPalette />

    <!-- 上传抽屉 -->
    <UploadDrawer />

    <!-- 图片查看器 -->
    <ImageViewer />

    <!-- 图片网格 -->
    <ImageGrid />
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useImageStore } from '@/stores/image'
import AppLayout from '@/components/layout/AppLayout.vue'
import CommandPalette from '@/components/search/CommandPalette.vue'
import UploadDrawer from '@/components/upload/UploadDrawer.vue'
import ImageViewer from '@/components/gallery/ImageViewer.vue'
import ImageGrid from '@/components/gallery/ImageGrid.vue'

const imageStore = useImageStore()

onMounted(async () => {
  // 加载图片列表
  if (imageStore.images.length === 0) {
    await imageStore.fetchImages()
  }
})
</script>
