<template>
  <aside
    :class="[
      'flex flex-col border-r border-gray-200 bg-white transition-all duration-300',
      uiStore.sidebarCollapsed ? 'w-16' : 'w-64',
    ]"
  >
    <!-- Logo区域 -->
    <div class="flex h-16 items-center justify-between border-b border-gray-200 px-4">
      <h1 v-if="!uiStore.sidebarCollapsed" class="text-xl font-semibold text-gray-900">
        影像库
      </h1>
      <button
        @click="uiStore.toggleSidebar"
        class="rounded-lg p-2 text-gray-600 hover:bg-gray-100"
      >
        <Bars3Icon v-if="!uiStore.sidebarCollapsed" class="h-5 w-5" />
        <Bars3Icon v-else class="h-5 w-5" />
      </button>
    </div>

    <!-- 导航菜单 -->
    <nav class="flex-1 overflow-y-auto p-4">
      <div class="space-y-1">
        <!-- 全部影像 -->
        <router-link
          to="/gallery"
          v-slot="{ isActive }"
          custom
        >
          <button
            @click="navigateTo('/gallery')"
            :class="[
              'flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
              isActive
                ? 'bg-blue-50 text-blue-600'
                : 'text-gray-700 hover:bg-gray-100',
            ]"
          >
            <PhotoIcon class="h-5 w-5 flex-shrink-0" />
            <span v-if="!uiStore.sidebarCollapsed">全部影像</span>
          </button>
        </router-link>

        <!-- 地点 (预留) -->
        <button
          disabled
          :class="[
            'flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium',
            'cursor-not-allowed text-gray-400',
          ]"
        >
          <MapPinIcon class="h-5 w-5 flex-shrink-0" />
          <span v-if="!uiStore.sidebarCollapsed">地点</span>
        </button>

        <!-- 人物 (预留) -->
        <button
          disabled
          :class="[
            'flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium',
            'cursor-not-allowed text-gray-400',
          ]"
        >
          <UserGroupIcon class="h-5 w-5 flex-shrink-0" />
          <span v-if="!uiStore.sidebarCollapsed">人物</span>
        </button>

        <!-- 时间线 (预留) -->
        <button
          disabled
          :class="[
            'flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium',
            'cursor-not-allowed text-gray-400',
          ]"
        >
          <CalendarIcon class="h-5 w-5 flex-shrink-0" />
          <span v-if="!uiStore.sidebarCollapsed">时间线</span>
        </button>
      </div>
    </nav>

    <!-- 底部信息 -->
    <div class="border-t border-gray-200 p-4">
      <div v-if="!uiStore.sidebarCollapsed" class="text-xs text-gray-500">
        <div class="flex items-center justify-between">
          <span>共 {{ imageStore.total }} 张图片</span>
        </div>
      </div>
      <div v-else class="flex justify-center">
        <span class="text-xs font-medium text-gray-600">{{ imageStore.total }}</span>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useUIStore } from '@/stores/ui'
import { useImageStore } from '@/stores/image'
import {
  PhotoIcon,
  MapPinIcon,
  UserGroupIcon,
  CalendarIcon,
  Bars3Icon,
} from '@heroicons/vue/24/outline'

const router = useRouter()
const uiStore = useUIStore()
const imageStore = useImageStore()

function navigateTo(path: string) {
  router.push(path)
}
</script>
