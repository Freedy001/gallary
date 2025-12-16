<template>
  <nav class="flex items-center gap-2 text-sm">
    <template v-for="(item, index) in items" :key="index">
      <!-- 分隔符 -->
      <ChevronRightIcon v-if="index > 0" class="h-4 w-4 text-gray-600 flex-shrink-0" />

      <!-- 链接项 -->
      <router-link
        v-if="item.to"
        :to="item.to"
        class="text-gray-400 hover:text-white transition-colors"
      >
        {{ item.label }}
      </router-link>

      <!-- 当前项 -->
      <span v-else class="text-white font-medium">
        {{ item.label }}
      </span>
    </template>
  </nav>
</template>

<script setup lang="ts">
import { ChevronRightIcon } from '@heroicons/vue/24/outline'

export interface BreadcrumbItem {
  label: string
  to?: string
}

defineProps<{
  items: BreadcrumbItem[]
}>()
</script>
