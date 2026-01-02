<template>
  <div
    ref="triggerRef"
    v-bind="$attrs"
    @mouseenter="onMouseEnter"
    @mouseleave="onMouseLeave"
  >
    <slot />
  </div>

  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-75 ease-out"
      enter-from-class="transform scale-95 opacity-0 translate-y-1"
      enter-to-class="transform scale-100 opacity-100 translate-y-0"
      leave-active-class="transition duration-75 ease-in"
      leave-from-class="transform scale-100 opacity-100 translate-y-0"
      leave-to-class="transform scale-95 opacity-0 translate-y-1"
    >
      <div
        v-if="isVisible && content"
        ref="tooltipRef"
        :style="tooltipStyle"
        class="fixed z-[9999] px-2.5 py-1.5 text-xs font-medium text-white bg-[#1A1A1A] rounded-lg shadow-[0_0_10px_rgba(0,0,0,0.5)] border border-white/10 pointer-events-none max-w-xs break-words"
      >
        {{ content }}
      </div>
    </Transition>
  </Teleport>
</template>

<script lang="ts" setup>
import {computed, nextTick, ref} from 'vue'

const props = defineProps<{
  content?: string | number
  placement?: 'top' | 'bottom'
  showOnlyIfTruncated?: boolean
}>()

const isVisible = ref(false)
const triggerRef = ref<HTMLElement | null>(null)
const tooltipRef = ref<HTMLElement | null>(null)
const position = ref({ top: 0, left: 0 })

const checkTruncation = (element: HTMLElement) => {
  // Helper to check a specific element
  const isOverflowing = (el: HTMLElement) => {
    // 1. Standard scroll check (allow for small rounding differences)
    if (el.scrollWidth > el.clientWidth + 1) return true

    // 2. Text truncation check (Range method)
    // This is more reliable for text-overflow: ellipsis cases where scrollWidth might not report overflow
    try {
      const range = document.createRange()
      range.selectNodeContents(el)
      const rangeWidth = range.getBoundingClientRect().width
      // Compare range width (content width) with element width
      // Add small buffer (0.5px) for subpixel rendering issues
      return rangeWidth > el.clientWidth + 0.5
    } catch (e) {
      return false
    }
  }

  // Check self
  if (isOverflowing(element)) return true

  // Check first child (wrapper pattern)
  const child = element.firstElementChild as HTMLElement
  if (child && isOverflowing(child)) return true

  return false
}

const onMouseEnter = async () => {
  if (!props.content) return
  if (props.showOnlyIfTruncated && triggerRef.value) {
    if (!checkTruncation(triggerRef.value)) return
  }

  // Update position before showing
  updatePosition()
  isVisible.value = true

  // Update again after render to ensure correct dimensions
  await nextTick()
  updatePosition()
}

const onMouseLeave = () => {
  isVisible.value = false
}

const updatePosition = () => {
  if (!triggerRef.value) return

  const rect = triggerRef.value.getBoundingClientRect()
  const scrollTop = window.scrollY || document.documentElement.scrollTop
  const scrollLeft = window.scrollX || document.documentElement.scrollLeft

  // Default to top center
  let top = rect.top + scrollTop - 10 // 10px spacing
  let left = rect.left + scrollLeft + (rect.width / 2)

  // If we have tooltip dimensions, we can center it
  if (tooltipRef.value) {
    const tooltipRect = tooltipRef.value.getBoundingClientRect()
    left -= tooltipRect.width / 2
    top -= tooltipRect.height

    // Boundary check - ensure it doesn't go off screen
    if (left < 4) left = 4
    if (left + tooltipRect.width > window.innerWidth - 4) {
      left = window.innerWidth - tooltipRect.width - 4
    }

    if (top < 4) {
      // Flip to bottom if not enough space on top
      top = rect.bottom + scrollTop + 10
    }
  } else {
    // Estimate
    top -= 30
  }

  position.value = { top, left }
}

const tooltipStyle = computed(() => ({
  top: `${position.value.top}px`,
  left: `${position.value.left}px`
}))
</script>
