<template>
  <div
    v-if="style"
    class="absolute z-50 border border-primary-500 bg-primary-500/20 pointer-events-none transition-none will-change-[width,height,top,left]"
    :style="style"
  ></div>
</template>

<script setup lang="ts">
import type {StyleValue} from 'vue'

defineProps<{
  style: StyleValue
}>()
</script>
