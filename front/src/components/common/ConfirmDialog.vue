<template>
  <Teleport to="body">
    <!-- Dialog Modal -->
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="dialogStore.state.visible"
        class="fixed inset-0 z-[100] flex items-center justify-center overflow-y-auto px-4 py-6 sm:px-0"
        role="dialog"
        aria-modal="true"
      >
        <!-- Backdrop -->
        <div
          class="fixed inset-0 bg-black/60 backdrop-blur-sm transition-opacity"
          @click="handleBackdropClick"
        ></div>

        <!-- Modal Panel -->
        <div
            class="relative w-full max-w-lg -translate-y-20 overflow-hidden rounded-2xl border border-white/10 bg-[#121212] p-6 text-left align-middle shadow-[0_0_40px_rgba(0,0,0,0.5)] transition-all"
        >
          <!-- Glow Effect -->
          <div
            class="absolute -top-20 -right-20 h-40 w-40 rounded-full blur-3xl opacity-20 pointer-events-none"
            :class="glowColor"
          ></div>

          <div class="relative flex gap-4">
            <!-- Icon -->
            <div
              class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full ring-1 ring-white/10"
              :class="iconBgColor"
            >
              <component :is="currentIcon" class="h-6 w-6" :class="iconColor" />
            </div>

            <!-- Content -->
            <div class="flex-1 pt-1">
              <h3 class="text-lg font-medium leading-6 text-white">
                {{ dialogStore.state.title }}
              </h3>
              <div class="mt-2">
                <p class="text-sm text-gray-400 leading-relaxed">
                  {{ dialogStore.state.message }}
                </p>
              </div>
            </div>
          </div>

          <!-- Actions -->
          <div class="mt-8 flex flex-wrap items-center justify-end gap-3">
            <button
              v-if="dialogStore.state.cancelText"
              @click="dialogStore.handleCancel"
              class="whitespace-nowrap rounded-xl px-4 py-2.5 text-sm font-medium text-gray-400 transition-colors hover:bg-white/5 hover:text-white focus:outline-none focus:ring-2 focus:ring-white/10"
            >
              {{ dialogStore.state.cancelText }}
            </button>

            <button
              v-if="dialogStore.state.thirdText"
              class="whitespace-nowrap rounded-xl border border-white/10 bg-white/5 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-white/10"
              @click="dialogStore.handleThird"
            >
              {{ dialogStore.state.thirdText }}
            </button>

            <button
              @click="dialogStore.handleConfirm"
              class="whitespace-nowrap rounded-xl px-5 py-2.5 text-sm font-medium text-white shadow-lg transition-all hover:scale-105 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-[#121212]"
              :class="confirmBtnClass"
            >
              {{ dialogStore.state.confirmText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

  <!-- Notifications (Toast) -->
  <Teleport to="body">
    <div class="fixed top-6 right-6 z-[110] flex flex-col items-end gap-3 w-full pointer-events-none px-4 sm:px-0">
      <TransitionGroup
        enter-active-class="transition duration-400 ease-[cubic-bezier(0.16,1,0.3,1)]"
        enter-from-class="opacity-0 translate-x-8 scale-95"
        enter-to-class="opacity-100 translate-x-0 scale-100"
        leave-active-class="transition duration-300 ease-in absolute"
        leave-from-class="opacity-100 translate-x-0 scale-100"
        leave-to-class="opacity-0 translate-x-8 scale-95"
        move-class="transition-all duration-500 ease-[cubic-bezier(0.16,1,0.3,1)]"
      >
        <div
          v-for="notification in dialogStore.notifications"
          :key="notification.id"
          class="pointer-events-auto relative w-80 overflow-hidden rounded-2xl p-0.5 shadow-[0_8px_30px_rgb(0,0,0,0.2)] backdrop-blur-xl transition-all hover:scale-[1.02] z-10"
          @mouseenter="pauseTimer(notification.id)"
          @mouseleave="resumeTimer(notification.id)"
        >
          <!-- 渐变边框背景 -->
          <div class="absolute inset-0 bg-gradient-to-br from-yellow-500/20 via-white/5 to-transparent opacity-50"></div>

          <!-- 内容容器 -->
          <div
            class="relative h-full w-full rounded-[14px] bg-gradient-to-br from-[#1a1a1a]/95 via-[#111]/90 to-black/95 p-4"
          >
            <!-- 顶部高光装饰 -->
            <div class="absolute inset-x-0 top-0 h-[1px] bg-gradient-to-r from-transparent via-yellow-400/30 to-transparent opacity-70"></div>

            <!-- 底部反光装饰 -->
            <div class="absolute inset-x-0 bottom-0 h-[1px] bg-gradient-to-r from-transparent via-yellow-400/10 to-transparent opacity-30"></div>

            <div class="relative flex items-start gap-3">
              <!-- 图标 -->
              <div
                class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl ring-1 ring-inset backdrop-blur-md shadow-inner"
                :class="[getIconBgColor(notification.type), getIconRingColor(notification.type)]"
              >
                <component
                  :is="getIcon(notification.type)"
                  class="h-5 w-5"
                  :class="getIconColor(notification.type)"
                />
              </div>

              <!-- 内容 -->
              <div class="flex-1 pt-0.5 min-w-0">
                <h3 v-if="notification.title" class="text-sm font-bold tracking-wide text-gray-100">{{ notification.title }}</h3>
                <p class="text-xs text-gray-400 leading-relaxed break-words font-medium" :class="{ 'mt-1': notification.title }">
                  {{ notification.message }}
                </p>
              </div>

              <!-- 关闭按钮 -->
              <button
                @click="dialogStore.removeNotification(notification.id)"
                class="group flex-shrink-0 -mr-1 -mt-1 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-white/10 hover:text-white"
              >
                <XMarkIcon class="h-3.5 w-3.5 transition-transform duration-300 group-hover:rotate-90" />
              </button>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import {computed, watch} from 'vue'
import {useDialogStore} from '@/stores/dialog.ts'
import type {DialogType, Notification} from '@/types/dialog.ts'
import {
  CheckCircleIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
  XCircleIcon,
  XMarkIcon
} from '@heroicons/vue/24/outline'

const dialogStore = useDialogStore()

// Notification Timer Logic
const timers = new Map<number, { id: any, remaining: number, start: number }>()

function startTimer(notification: Notification) {
  const duration = notification.duration
  if (duration && duration > 0) {
    const timerId = setTimeout(() => {
      dialogStore.removeNotification(notification.id)
      timers.delete(notification.id)
    }, duration)

    timers.set(notification.id, {
      id: timerId,
      remaining: duration,
      start: Date.now()
    })
  }
}

function pauseTimer(id: number) {
  const timer = timers.get(id)
  if (timer) {
    clearTimeout(timer.id)
    timer.remaining -= Date.now() - timer.start
    timers.delete(id)
    // Store the paused state back in the map, but with no timer ID
    timers.set(id, { ...timer, id: null })
  }
}

function resumeTimer(id: number) {
  const timer = timers.get(id)
  if (timer && timer.remaining > 0) {
    const timerId = setTimeout(() => {
      dialogStore.removeNotification(id)
      timers.delete(id)
    }, timer.remaining)

    timers.set(id, {
      id: timerId,
      remaining: timer.remaining,
      start: Date.now()
    })
  }
}

// Watch for new notifications
watch(() => dialogStore.notifications, (newVal) => {
  // Iterate new values and start timer if not already tracked
  newVal.forEach(n => {
    // Only start timer if it's not already in the map (avoid restarting paused timers or duplicates)
    if (!timers.has(n.id)) {
      startTimer(n)
    }
  })

  // Clean up removed timers
  // Iterate active timers and remove those not in the new list
  for (const [id, timer] of timers.entries()) {
    if (!newVal.some(n => n.id === id)) {
      if (timer.id) {
        clearTimeout(timer.id)
      }
      timers.delete(id)
    }
  }
}, { deep: true })

// Helper functions for both Dialog and Notifications
const getIcon = (type: DialogType) => {
  switch (type) {
    case 'success': return CheckCircleIcon
    case 'error': return XCircleIcon
    case 'warning': return ExclamationTriangleIcon
    case 'confirm': return InformationCircleIcon
    default: return InformationCircleIcon
  }
}

const getIconColor = (type: DialogType) => {
  switch (type) {
    case 'success': return 'text-emerald-400'
    case 'error': return 'text-red-400'
    case 'warning': return 'text-amber-400'
    case 'confirm': return 'text-primary-400'
    default: return 'text-primary-400'
  }
}

const getIconBgColor = (type: DialogType) => {
  switch (type) {
    case 'success': return 'bg-emerald-500/10'
    case 'error': return 'bg-red-500/10'
    case 'warning': return 'bg-amber-500/10'
    case 'confirm': return 'bg-primary-500/10'
    default: return 'bg-primary-500/10'
  }
}

const getIconRingColor = (type: DialogType) => {
  switch (type) {
    case 'success': return 'ring-emerald-500/20'
    case 'error': return 'ring-red-500/20'
    case 'warning': return 'ring-amber-500/20'
    case 'confirm': return 'ring-primary-500/20'
    default: return 'ring-primary-500/20'
  }
}

const getGlowColor = (type: DialogType) => {
  switch (type) {
    case 'success': return 'bg-emerald-500'
    case 'error': return 'bg-red-500'
    case 'warning': return 'bg-amber-500'
    case 'confirm': return 'bg-primary-500'
    default: return 'bg-primary-500'
  }
}

// Computed for current Dialog
const currentIcon = computed(() => getIcon(dialogStore.state.type || 'info'))
const glowColor = computed(() => getGlowColor(dialogStore.state.type || 'info'))

const iconBgColor = computed(() => {
  switch (dialogStore.state.type) {
    case 'success': return 'bg-emerald-500/10'
    case 'error': return 'bg-red-500/10'
    case 'warning': return 'bg-amber-500/10'
    case 'confirm': return 'bg-primary-500/10'
    default: return 'bg-primary-500/10'
  }
})

const iconColor = computed(() => getIconColor(dialogStore.state.type || 'info'))

const confirmBtnClass = computed(() => {
  switch (dialogStore.state.type) {
    case 'error':
      return 'bg-red-500 hover:bg-red-600 hover:shadow-red-500/20 focus:ring-red-500'
    case 'warning':
      return 'bg-amber-600 hover:bg-amber-700 hover:shadow-amber-500/20 focus:ring-amber-600'
    case 'success':
      return 'bg-emerald-600 hover:bg-emerald-700 hover:shadow-emerald-500/20 focus:ring-emerald-600'
    default:
      return 'bg-primary-600 hover:bg-primary-500 hover:shadow-primary-500/20 focus:ring-primary-500'
  }
})

function handleBackdropClick() {
  // Only close on backdrop click if it's not a confirm dialog (optional choice)
  // For now, let's make it cancel
  dialogStore.handleCancel()
}
</script>
