<template>
  <div :class="square ? 'aspect-square' : ''" class="relative group">
    <ImageCard
        :image="image"
        :square="square"
        :load-priority="loadPriority"
    />

    <!-- 选择模式遮罩 -->
    <div
        v-if="isSelectionMode"
        :class="[
            square ? 'rounded-2xl' : 'rounded-lg',
            isSelected
              ? 'bg-primary-500/20 ring-2 ring-primary-500'
              : 'hover:bg-black/10'
          ]"
        class="absolute inset-0 cursor-pointer transition-colors"
    >
      <div class="absolute top-2 right-2">
        <div
            :class="[
                isSelected
                  ? 'border-primary-500 bg-primary-500 text-white'
                  : 'border-white/70 bg-black/20 hover:bg-black/40'
              ]"
            class="flex h-6 w-6 items-center justify-center rounded-full border-2 transition-colors"
        >
          <CheckIcon v-if="isSelected" class="h-4 w-4"/>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type {Image} from '@/types'
import {CheckIcon,} from '@heroicons/vue/24/outline'
import ImageCard from './ImageCard.vue'

defineProps<{
  image: Image | null | undefined
  isSelected: boolean
  isSelectionMode: boolean
  square?: boolean
  loadPriority?: number
}>()
</script>

