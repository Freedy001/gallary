<template>
  <div
      v-if="modelValue"
      :style="style"
      class="fixed z-50 min-w-[200px] w-max !rounded-[24px]"
      @contextmenu.prevent
  >
    <LiquidGlassCard :hover-effect="false" content-class="p-1 pt-2">
      <slot></slot>
    </LiquidGlassCard>
  </div>

</template>

<script setup lang="ts">
import {computed, onMounted, onUnmounted, ref, watch} from 'vue'
import LiquidGlassCard from '@/components/widgets/common/LiquidGlassCard.vue'

const props = defineProps<{
  modelValue: boolean
  x: number
  y: number
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()
ref<HTMLElement | null>(null);
const adjustedX = ref(props.x)
const adjustedY = ref(props.y)

const style = computed(() => ({
  top: `${adjustedY.value}px`,
  left: `${adjustedX.value}px`,
}))

const close = () => {
  emit('update:modelValue', false)
}

const handleClickOutside = () => {
  // Simple check: close on any click. Since the menu handles its own clicks,
  // clicks inside will propagate and trigger actions, then we close.
  // Or we can rely on the parent to close it, but standard behavior is close on click anywhere.
  close()
}

// Watch for opening to adjust position (optional, skipping for simplicity unless requested)
watch(() => [props.x, props.y], () => {
  adjustedX.value = props.x
  adjustedY.value = props.y
})

// Close on scroll
const handleScroll = () => {
  if (props.modelValue) {
    close()
  }
}

onMounted(() => {
  // Using window click to close menu
  window.addEventListener('click', handleClickOutside)
  window.addEventListener('scroll', handleScroll, true)
})

onUnmounted(() => {
  window.removeEventListener('click', handleClickOutside)
  window.removeEventListener('scroll', handleScroll, true)
})
</script>
