<template>
  <AppLayout>
    <template #header>
      <TopBar/>
    </template>

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
import {onMounted} from 'vue'
import {useImageStore} from '@/stores/image'
import {useUIStore} from '@/stores/ui'
import AppLayout from '@/components/layout/AppLayout.vue'
import CommandPalette from '@/components/search/CommandPalette.vue'
import ImageGrid from '@/components/gallery/ImageGrid.vue'
import Timeline from '@/components/gallery/Timeline.vue'
import type {Image, Pageable} from "@/types";
import {imageApi} from "@/api/image.ts";
import TopBar from "@/components/layout/TopBar.vue";

const imageStore = useImageStore()
const uiStore = useUIStore()

onMounted(async () => {
  const pageSize = uiStore.imagePageSize
  await imageStore.refreshImages(async (page: number, size: number): Promise<Pageable<Image>> => (await imageApi.getList(page, size)).data, pageSize)

  // 加载图片列表
  if (imageStore.images.length === 0) {
    await imageStore.fetchImages(1, pageSize)
  }
})
</script>
