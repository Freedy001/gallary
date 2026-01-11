<template>
  <Teleport to="body">
    <div
        v-if="modelValue"
        ref="menuRef"
        :style="style"
        class="fixed z-100 min-w-[200px] w-max rounded-3xl!"
        @contextmenu.prevent
    >
      <LiquidGlassCard :hover-effect="false" content-class="p-1 pt-2">
        <slot></slot>
      </LiquidGlassCard>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
import {computed, onMounted, onUnmounted, ref, watch} from 'vue'
import LiquidGlassCard from '@/components/common/LiquidGlassCard.vue'

const props = defineProps<{
  modelValue: boolean
  x: number
  y: number
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

const menuRef = ref<HTMLElement | null>(null)
const adjustedX = ref(props.x)
const adjustedY = ref(props.y)

const style = computed(() => ({
  top: `${adjustedY.value}px`,
  left: `${adjustedX.value}px`,
}))

const close = () => {
  emit('update:modelValue', false)
}

const handleClickOutside = (e: MouseEvent) => {
  // 检查点击是否在菜单外部
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    close()
  }
}

let clickListenerTimer: ReturnType<typeof setTimeout> | null = null

// Watch for opening to adjust position (optional, skipping for simplicity unless requested)
watch(() => [props.x, props.y], () => {
  adjustedX.value = props.x
  adjustedY.value = props.y
})

// 监听菜单打开/关闭状态
watch(() => props.modelValue, (isVisible) => {
  if (isVisible) {
    // 菜单打开时，延迟添加点击监听器，避免立即被关闭
    if (clickListenerTimer) clearTimeout(clickListenerTimer)
    clickListenerTimer = setTimeout(() => {
      // 使用 capture 阶段捕获点击事件
      document.addEventListener('click', handleClickOutside, true)
    }, 100)
  } else {
    // 菜单关闭时移除监听器
    document.removeEventListener('click', handleClickOutside, true)
    if (clickListenerTimer) {
      clearTimeout(clickListenerTimer)
      clickListenerTimer = null
    }
  }
})

// Close on scroll
const handleScroll = () => {
  if (props.modelValue) {
    close()
  }
}

onMounted(() => {
  // 不再在这里添加全局点击监听器，而是在 watch 中添加
  window.addEventListener('scroll', handleScroll, true)
})

onUnmounted(() => {
  if (clickListenerTimer) clearTimeout(clickListenerTimer)
  document.removeEventListener('click', handleClickOutside, true)
  window.removeEventListener('scroll', handleScroll, true)
})
</script>
