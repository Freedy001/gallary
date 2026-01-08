<template>
  <div
    class="px-3 py-2.5 mx-1 my-0.5 text-sm cursor-pointer flex items-center gap-3 transition-all duration-200 rounded-lg group"
    :class="[
      danger
        ? 'text-red-400 hover:bg-red-500/10 hover:text-red-300'
        : 'text-gray-300 hover:bg-white/10 hover:text-white'
    ]"
    @click="handleClick"
  >
    <component
      v-if="icon"
      :is="icon"
      class="w-4 h-4 transition-transform group-hover:scale-110"
      :class="danger ? 'text-red-400 group-hover:text-red-300' : 'text-gray-400 group-hover:text-white'"
    />
    <span class="font-medium tracking-wide"><slot></slot></span>
  </div>
</template>

<script setup lang="ts">
import type {Component} from 'vue'

defineProps<{
  icon?: Component
  danger?: boolean
}>()

const emit = defineEmits(['click'])

const handleClick = (e: MouseEvent) => {
  emit('click', e)
}
</script>
