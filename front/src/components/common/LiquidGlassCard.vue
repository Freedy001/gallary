<template>
  <div
    ref="glassCardRef"
    class="liquid-glass-card"
    :class="{ 'with-hover': hoverEffect }"
  >
    <!-- Dynamic Backdrop Layer -->
    <div
        v-if="targetImage && targetElement"
        class="absolute inset-0 overflow-hidden rounded-[24px] z-0 pointer-events-none"
    >
        <div
            class="absolute top-0 left-0 origin-top-left backdrop-filter-layer"
            :style="backdropStyle"
        ></div>
    </div>

    <!-- Glass Highlight Layer -->
    <div class="glass-highlight"></div>

    <!-- Content Layer -->
    <div class="relative z-10" :class="contentClass">
      <slot></slot>
    </div>

    <!-- SVG Filter Definition (Global usage, defined once here or use external ID if defined elsewhere) -->
    <!--
         Note: If multiple cards are present, we only need the filter defined once globally.
         However, keeping it here ensures self-containment.
         The ID 'dispersion-filter' must be unique if we want strict correctness,
         but SVG filters with same ID usually just use the first one found.
    -->
    <svg style="position: absolute; width: 0; height: 0; pointer-events: none;">
      <defs>
        <filter id="dispersion-filter" x="-20%" y="-20%" width="140%" height="140%">
          <feTurbulence type="fractalNoise" baseFrequency="0.02" numOctaves="3" result="noise" />
          <feDisplacementMap in="SourceGraphic" in2="noise" scale="15" result="distorted" />
          <feGaussianBlur in="distorted" stdDeviation="5" result="blurred" />
          <!-- Red Channel -->
          <feColorMatrix type="matrix" in="blurred" result="red"
            values="1 0 0 0 0
                    0 0 0 0 0
                    0 0 0 0 0
                    0 0 0 1 0" />
          <feOffset in="red" dx="-3" dy="0" result="red_offset" />
          <!-- Green Channel -->
          <feColorMatrix type="matrix" in="blurred" result="green"
            values="0 0 0 0 0
                    0 1 0 0 0
                    0 0 0 0 0
                    0 0 0 1 0" />
          <!-- Blue Channel -->
          <feColorMatrix type="matrix" in="blurred" result="blue"
            values="0 0 0 0 0
                    0 0 0 0 0
                    0 0 1 0 0
                    0 0 0 1 0" />
          <feOffset in="blue" dx="3" dy="0" result="blue_offset" />
          <!-- Blend -->
          <feBlend mode="screen" in="red_offset" in2="green" result="rg" />
          <feBlend mode="screen" in="rg" in2="blue_offset" result="rgb" />
        </filter>
      </defs>
    </svg>
  </div>
</template>

<script setup lang="ts">
import { ref, onUnmounted, watch } from 'vue'
import { useRafFn } from '@vueuse/core'

const props = withDefaults(defineProps<{
  targetElement?: HTMLElement | null
  targetImage?: string
  hoverEffect?: boolean
  contentClass?: string
}>(), {
  hoverEffect: true,
  contentClass: 'p-4'
})

const glassCardRef = ref<HTMLElement>()
const backdropStyle = ref({})

// Sync Logic
const { pause, resume } = useRafFn(() => {
  if (!props.targetElement || !props.targetImage || !glassCardRef.value) {
      // Clear style if targets missing
      if (Object.keys(backdropStyle.value).length > 0) backdropStyle.value = {}
      return
  }

  // If targetElement is an IMG, we can use its geometry directly
  // Or if it is a container, we assume it positions the image.
  // The logic from ImageViewer assumed targetElement IS the image tag.
  const targetRect = props.targetElement.getBoundingClientRect()
  const cardRect = glassCardRef.value.getBoundingClientRect()

  // Calculate relative position
  const left = targetRect.left - cardRect.left
  const top = targetRect.top - cardRect.top
  const width = targetRect.width
  const height = targetRect.height

  backdropStyle.value = {
    backgroundImage: `url('${props.targetImage}')`,
    backgroundSize: `${width}px ${height}px`,
    backgroundPosition: `${left}px ${top}px`,
    backgroundRepeat: 'no-repeat',
    width: `${cardRect.width}px`,
    height: `${cardRect.height}px`,
  }
})

watch(() => [props.targetElement, props.targetImage], ([el, img]) => {
    if (el && img) resume()
    else pause()
}, { immediate: true })

onUnmounted(() => pause())
</script>

<style scoped>
.liquid-glass-card {
  position: relative;
  border-radius: 24px;
  background: linear-gradient(
      135deg,
      rgba(255, 255, 255, 0.15) 0%,
      rgba(255, 255, 255, 0.05) 100%
  );
  border: 1px solid rgba(255, 255, 255, 0.18);
  box-shadow:
      0 8px 32px 0 rgba(0, 0, 0, 0.37),
      inset 1px 0 1px rgba(255, 100, 100, 0.25),
      inset -1px 0 1px rgba(100, 200, 255, 0.25),
      inset 0 1px 0 rgba(255, 255, 255, 0.4);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  overflow: hidden;
  transition: transform 0.4s cubic-bezier(0.25, 0.8, 0.25, 1), box-shadow 0.4s cubic-bezier(0.25, 0.8, 0.25, 1);
}

.with-hover:hover {
  transform: scale(1.02) translateY(-2px);
  box-shadow:
      0 15px 35px rgba(0, 0, 0, 0.4),
      inset 2px 0 2px rgba(255, 100, 100, 0.4),
      inset -2px 0 2px rgba(100, 200, 255, 0.4),
      inset 0 1px 0 rgba(255, 255, 255, 0.5);
  background: linear-gradient(
      135deg,
      rgba(255, 255, 255, 0.25) 0%,
      rgba(255, 255, 255, 0.1) 100%
  );
}

.backdrop-filter-layer {
  transform-origin: 0 0;
  filter: url(#dispersion-filter);
}

.glass-highlight {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 50%;
  background: linear-gradient(
      to bottom,
      rgba(255, 255, 255, 0.1) 0%,
      rgba(255, 255, 255, 0) 100%
  );
  pointer-events: none;
  z-index: 5;
  border-radius: 24px 24px 0 0;
}
</style>
