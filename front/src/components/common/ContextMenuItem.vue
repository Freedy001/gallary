<template>
  <div
    class="px-4 py-2 text-sm cursor-pointer flex items-center gap-2 transition-colors duration-150"
    :class="[
      danger
        ? 'text-red-600 hover:bg-red-50 font-bold'
        : 'text-white hover:bg-gray-100/50  font-bold'
    ]"
    @click="handleClick"
  >
    <component
      v-if="icon"
      :is="icon"
      class="w-4 h-4"
      :class="danger ? 'text-red-500' : 'text-white'"
    />
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  icon?: Component
  danger?: boolean
}>()

const emit = defineEmits(['click'])

const handleClick = (e: MouseEvent) => {
  emit('click', e)
}
</script>
