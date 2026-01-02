<template>
  <aside
    :class="[
      'flex flex-col transition-all duration-500 ease-[cubic-bezier(0.16,1,0.3,1)]',
      'backdrop-blur-xl bg-black/40 border-r border-white/5',
      uiStore.sidebarCollapsed ? 'w-20' : 'w-64',
    ]"
  >
    <!-- Logo区域 -->
    <div class="flex h-20 items-center justify-between px-6" style="user-select: none">
      <div v-if="!uiStore.sidebarCollapsed" class="flex items-center gap-2 overflow-hidden whitespace-nowrap">
        <div class="h-8 w-8 rounded-lg bg-gradient-to-br from-primary-500/80 to-primary-700/80 shadow-[0_0_15px_rgba(139,92,246,0.2)] flex items-center justify-center border border-white/10">
           <svg class="w-5 h-5 text-white" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
             <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
             <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
             <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
           </svg>
        </div>
        <h1 class="text-xl font-bold text-white tracking-wide font-sans drop-shadow-sm">
          GALLERY
        </h1>
      </div>
      <!-- 收起时的 Logo -->
      <div v-else class="w-full flex justify-center">
         <div class="h-8 w-8 rounded-lg bg-gradient-to-br from-primary-500/80 to-primary-700/80 shadow-[0_0_10px_rgba(139,92,246,0.2)] flex items-center justify-center border border-white/10">
             <svg class="w-5 h-5 text-white" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
               <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
             </svg>
         </div>
      </div>

      <button
        v-if="!uiStore.sidebarCollapsed"
        @click="uiStore.toggleSidebar"
        class="rounded-full p-1.5 text-gray-500 hover:bg-white/10 hover:text-white transition-colors"
      >
        <ChevronLeftIcon class="h-4 w-4" />
      </button>
    </div>

    <!-- 收起模式下的展开按钮 -->
    <div v-if="uiStore.sidebarCollapsed" class="flex justify-center pb-4">
        <button
        @click="uiStore.toggleSidebar"
        class="rounded-full p-2 text-gray-500 hover:bg-white/10 hover:text-white transition-colors"
      >
        <Bars3Icon class="h-5 w-5" />
      </button>
    </div>

    <!-- 导航菜单 -->
    <nav class="flex-1 overflow-y-auto px-3 py-2 scrollbar-none">
      <div class="space-y-2">
        <!-- 全部影像 -->
        <router-link
          to="/gallery"
          v-slot="{ isActive }"
          custom
        >
          <button
            @click="navigateTo('/gallery')"
            :class="[
              'group flex w-full items-center gap-3 rounded-xl px-3 py-3 text-sm font-medium transition-all duration-300 relative overflow-hidden',
              isActive
                ? 'text-white bg-white/10 shadow-[inset_0_1px_0_0_rgba(255,255,255,0.1)]'
                : 'text-gray-400 hover:text-gray-100 hover:bg-white/5',
              uiStore.sidebarCollapsed ? 'justify-center' : ''
            ]"
          >
            <!-- Active Indicator Glow -->
            <div v-if="isActive" class="absolute left-0 top-0 bottom-0 w-1 bg-primary-500 shadow-[0_0_10px_rgba(139,92,246,0.6)]"></div>

            <PhotoIcon :class="['h-5 w-5 flex-shrink-0 transition-transform duration-300', isActive ? 'text-primary-400 scale-110' : 'group-hover:scale-110']" />
            <span v-if="!uiStore.sidebarCollapsed" class="tracking-wide">全部影像</span>

            <!-- Tooltip for collapsed state could go here -->
          </button>
        </router-link>

        <!-- 相册 -->
        <router-link
            to="/gallery/albums"
            v-slot="{ isActive }"
            custom
        >
          <button
              @click="navigateTo('/gallery/albums')"
              :class="[
              'group flex w-full items-center gap-3 rounded-xl px-3 py-3 text-sm font-medium transition-all duration-300 relative overflow-hidden',
              isActive || route.path.startsWith('/gallery/albums')
                ? 'text-white bg-white/10 shadow-[inset_0_1px_0_0_rgba(255,255,255,0.1)]'
                : 'text-gray-400 hover:text-gray-100 hover:bg-white/5',
               uiStore.sidebarCollapsed ? 'justify-center' : ''
            ]"
          >
            <div v-if="isActive || route.path.startsWith('/gallery/albums')" class="absolute left-0 top-0 bottom-0 w-1 bg-primary-500 shadow-[0_0_10px_rgba(139,92,246,0.6)]"></div>
            <RectangleStackIcon :class="['h-5 w-5 flex-shrink-0 transition-transform duration-300', (isActive || route.path.startsWith('/gallery/albums')) ? 'text-primary-400 scale-110' : 'group-hover:scale-110']" />
            <span v-if="!uiStore.sidebarCollapsed" class="tracking-wide">相册</span>
          </button>
        </router-link>

        <!-- 分享管理 -->
        <router-link
          to="/gallery/share"
          v-slot="{ isActive }"
          custom
        >
          <button
            @click="navigateTo('/gallery/share')"
            :class="[
              'group flex w-full items-center gap-3 rounded-xl px-3 py-3 text-sm font-medium transition-all duration-300 relative overflow-hidden',
              isActive
                ? 'text-white bg-white/10 shadow-[inset_0_1px_0_0_rgba(255,255,255,0.1)]'
                : 'text-gray-400 hover:text-gray-100 hover:bg-white/5',
               uiStore.sidebarCollapsed ? 'justify-center' : ''
            ]"
          >
            <div v-if="isActive" class="absolute left-0 top-0 bottom-0 w-1 bg-primary-500 shadow-[0_0_10px_rgba(139,92,246,0.6)]"></div>
            <ShareIcon :class="['h-5 w-5 flex-shrink-0 transition-transform duration-300', isActive ? 'text-primary-400 scale-110' : 'group-hover:scale-110']" />
            <span v-if="!uiStore.sidebarCollapsed" class="tracking-wide">分享管理</span>
          </button>
        </router-link>


        <!-- 地点 -->
        <router-link
            to="/gallery/location"
            v-slot="{ isActive }"
            custom
        >
          <button
              @click="navigateTo('/gallery/location')"
              :class="[
              'group flex w-full items-center gap-3 rounded-xl px-3 py-3 text-sm font-medium transition-all duration-300 relative overflow-hidden',
              isActive
                ? 'text-white bg-white/10 shadow-[inset_0_1px_0_0_rgba(255,255,255,0.1)]'
                : 'text-gray-400 hover:text-gray-100 hover:bg-white/5',
               uiStore.sidebarCollapsed ? 'justify-center' : ''
            ]"
          >
            <div v-if="isActive" class="absolute left-0 top-0 bottom-0 w-1 bg-primary-500 shadow-[0_0_10px_rgba(139,92,246,0.6)]"></div>
            <MapPinIcon :class="['h-5 w-5 flex-shrink-0 transition-transform duration-300', isActive ? 'text-primary-400 scale-110' : 'group-hover:scale-110']" />
            <span v-if="!uiStore.sidebarCollapsed" class="tracking-wide">地点足迹</span>
          </button>
        </router-link>

        <!-- 分隔线 -->
        <div class="my-4 border-t border-white/5 mx-2"></div>

        <!-- 人物 (预留) -->
        <button
          disabled
          :class="[
            'flex w-full items-center gap-3 rounded-xl px-3 py-3 text-sm font-medium transition-colors opacity-40',
            'cursor-not-allowed text-gray-500',
             uiStore.sidebarCollapsed ? 'justify-center' : ''
          ]"
        >
          <UserGroupIcon class="h-5 w-5 flex-shrink-0" />
          <span v-if="!uiStore.sidebarCollapsed">智能人物</span>
        </button>

        <!-- 时间线 (预留) -->
        <button
          disabled
          :class="[
            'flex w-full items-center gap-3 rounded-xl px-3 py-3 text-sm font-medium transition-colors opacity-40',
            'cursor-not-allowed text-gray-500',
             uiStore.sidebarCollapsed ? 'justify-center' : ''
          ]"
        >
          <CalendarIcon class="h-5 w-5 flex-shrink-0" />
          <span v-if="!uiStore.sidebarCollapsed">时光轴</span>
        </button>

        <!-- 分隔线 -->
        <div class="my-4 border-t border-white/5 mx-2"></div>

        <!-- 最近删除 -->
        <router-link
          to="/gallery/trash"
          v-slot="{ isActive }"
          custom
        >
          <button
            @click="navigateTo('/gallery/trash')"
            :class="[
              'group flex w-full items-center gap-3 rounded-xl px-3 py-3 text-sm font-medium transition-all duration-300 relative overflow-hidden',
              isActive
                ? 'text-white bg-white/10 shadow-[inset_0_1px_0_0_rgba(255,255,255,0.1)]'
                : 'text-gray-400 hover:text-gray-100 hover:bg-white/5',
               uiStore.sidebarCollapsed ? 'justify-center' : ''
            ]"
          >
            <div v-if="isActive" class="absolute left-0 top-0 bottom-0 w-1 bg-primary-500 shadow-[0_0_10px_rgba(139,92,246,0.6)]"></div>
            <TrashIcon :class="['h-5 w-5 flex-shrink-0 transition-transform duration-300', isActive ? 'text-primary-400 scale-110' : 'group-hover:scale-110']" />
            <span v-if="!uiStore.sidebarCollapsed" class="tracking-wide">最近删除</span>
          </button>
        </router-link>

        <!-- 分隔线 -->
        <div class="my-4 border-t border-white/5 mx-2"></div>

        <!-- 系统设置 -->
        <router-link
          to="/gallery/settings"
          v-slot="{ isActive }"
          custom
        >
          <button
            @click="navigateTo('/gallery/settings')"
            :class="[
              'group flex w-full items-center gap-3 rounded-xl px-3 py-3 text-sm font-medium transition-all duration-300 relative overflow-hidden',
              isActive
                ? 'text-white bg-white/10 shadow-[inset_0_1px_0_0_rgba(255,255,255,0.1)]'
                : 'text-gray-400 hover:text-gray-100 hover:bg-white/5',
               uiStore.sidebarCollapsed ? 'justify-center' : ''
            ]"
          >
            <div v-if="isActive" class="absolute left-0 top-0 bottom-0 w-1 bg-primary-500 shadow-[0_0_10px_rgba(139,92,246,0.6)]"></div>
            <Cog6ToothIcon :class="['h-5 w-5 flex-shrink-0 transition-transform duration-300', isActive ? 'text-primary-400 scale-110' : 'group-hover:scale-110']" />
            <span v-if="!uiStore.sidebarCollapsed" class="tracking-wide">系统设置</span>
          </button>
        </router-link>
      </div>
    </nav>

    <!-- 底部信息 -->
    <div class="border-t border-white/5 p-4 bg-black/20 backdrop-blur-md">
      <UploadDrawer :collapsed="uiStore.sidebarCollapsed" />

      <div v-if="!uiStore.sidebarCollapsed" class="mt-4 text-xs text-gray-500 font-mono tracking-wider flex items-center justify-between">
        <span>总影像</span>
        <span class="text-primary-400 font-bold">{{ notificationStore.imageCount }}</span>
      </div>
      <div v-else class="mt-4 flex justify-center">
        <span class="text-[10px] font-bold text-primary-500/70">{{ notificationStore.imageCount }}</span>
      </div>

      <!-- 存储容量 -->
      <div class="mt-3">
        <StorageUsage :collapsed="uiStore.sidebarCollapsed" />
      </div>

      <!-- AI 队列状态 - 仅在有任务时显示 -->
      <Transition
        enter-active-class="transition-all duration-300 ease-out"
        leave-active-class="transition-all duration-200 ease-in"
        enter-from-class="opacity-0 -translate-y-2 scale-95"
        enter-to-class="opacity-100 translate-y-0 scale-100"
        leave-from-class="opacity-100 translate-y-0 scale-100"
        leave-to-class="opacity-0 -translate-y-2 scale-95"
      >
        <div v-if="hasAITasks" class="mt-3">
          <AIQueueStatus :collapsed="uiStore.sidebarCollapsed"/>
        </div>
      </Transition>
    </div>
  </aside>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {useUIStore} from '@/stores/ui'
import {useNotificationStore} from '@/stores/notification'
import UploadDrawer from '@/components/upload/UploadDrawer.vue'
import StorageUsage from '@/components/widgets/StorageUsage.vue'
import AIQueueStatus from '@/components/widgets/AIQueueStatus.vue'
import {
  Bars3Icon,
  CalendarIcon,
  ChevronLeftIcon,
  Cog6ToothIcon,
  MapPinIcon,
  PhotoIcon,
  RectangleStackIcon,
  ShareIcon,
  TrashIcon,
  UserGroupIcon,
} from '@heroicons/vue/24/outline'

const router = useRouter()
const route = useRoute()
const uiStore = useUIStore()
const notificationStore = useNotificationStore()

// 判断是否有 AI 任务（pending 或 failed，或有队列正在处理中）
const hasAITasks = computed(() => {
  const status = notificationStore.aiQueueStatus
  if (!status) return false
  const hasProcessing = status.queues?.some(q => q.status === 'processing') || false
  return (status.total_pending > 0 || hasProcessing || status.total_failed > 0)
})

function navigateTo(path: string) {
  router.push(path)
}

</script>