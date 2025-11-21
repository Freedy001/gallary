<template>
  <Transition
      enter-active-class="transition duration-100 linear"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100 "
      leave-active-class="transition duration-300 linear"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
  >
    <div
        v-if="formattedDate"
        class="pointer-events-none absolute left-4 top-4 z-30"
    >
      <liquid-glass-card>
        <!-- 仿 iOS 悬浮毛玻璃效果 -->
        <div
            class="flex flex-col items-start px-5 py-2.5 rounded-[20px] transition-all duration-300">
          <!-- 年份 -->
          <div class="text-[11px] font-semibold text-white/80 leading-none mb-0.5 uppercase tracking-wide">
            {{ formattedDate.year }}
          </div>
          <!-- 月日 -->
          <div class="flex items-baseline gap-1.5">
            <h2 class="text-xl font-bold text-white leading-none tracking-tight font-display">
              {{ formattedDate.monthDay }}
            </h2>
            <span class="text-xs font-medium text-white/90">
            {{ formattedDate.weekday }}
          </span>
          </div>
        </div>
      </liquid-glass-card>

    </div>
  </Transition>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useUIStore} from '@/stores/ui'
import LiquidGlassCard from "@/components/common/LiquidGlassCard.vue";

const uiStore = useUIStore()

const formattedDate = computed(() => {
  if (!uiStore.activeDate) return null

  try {
    const date = new Date(uiStore.activeDate)
    if (isNaN(date.getTime())) return null

    const year = date.getFullYear()
    const month = date.getMonth() + 1
    const day = date.getDate()
    const weekday = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][date.getDay()]

    return {
      year: `${year}年`,
      monthDay: `${month}月${day}日`,
      weekday
    }
  } catch (e) {
    return null
  }
})
</script>
