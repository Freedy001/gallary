<template>
  <AppLayout>
    <template #header>
      <header
          class="sticky top-0 z-30 flex h-16 w-full items-center justify-between border-b border-white/5 bg-black/20 backdrop-blur-xl px-6 transition-all duration-300">
        <h1 class="text-xl font-bold tracking-wide text-white/90 font-display">系统设置</h1>
      </header>
    </template>

    <template #default>
      <div class="p-6 min-h-[calc(100vh-4rem)]">
        <div class="max-w-4xl mx-auto">
          <!-- Tab 导航 -->
          <div class="flex gap-2 mb-8 p-1 rounded-xl bg-white/5 ring-1 ring-white/10 w-fit">
            <button
                v-for="tab in tabs"
                :key="tab.id"
                :class="[
                'px-5 py-2.5 rounded-lg text-sm font-medium transition-all duration-300',
                uiStore.settingActiveTab === tab.id
                  ? 'bg-primary-500/20 text-primary-400 ring-1 ring-primary-500/30 shadow-[0_0_15px_rgba(139,92,246,0.2)]'
                  : 'text-gray-400 hover:text-white hover:bg-white/5'
              ]"
                @click="uiStore.settingActiveTab = tab.id"
            >
              <component :is="tab.icon" class="h-4 w-4 inline-block mr-2"/>
              {{ tab.name }}
            </button>
          </div>

          <!-- 安全设置 Tab -->
          <div v-if="uiStore.settingActiveTab === 'security'" class="space-y-6">
            <SecuritySettings />
          </div>

          <!-- 存储设置 Tab -->
          <div v-if="uiStore.settingActiveTab === 'storage'" class="space-y-6">
            <StorageSettings />
          </div>

          <!-- 清理策略 Tab -->
          <div v-if="uiStore.settingActiveTab === 'cleanup'" class="space-y-6">
            <CleanupSettings />
          </div>

          <!-- AI 设置 Tab -->
          <div v-if="uiStore.settingActiveTab === 'ai'" class="space-y-6">
            <AISettings />
          </div>
        </div>
      </div>
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import SecuritySettings from '@/components/settings/SecuritySettings.vue'
import StorageSettings from '@/components/settings/StorageSettings.vue'
import CleanupSettings from '@/components/settings/CleanupSettings.vue'
import AISettings from '@/components/settings/AISettings.vue'
import {CloudIcon, ShieldCheckIcon, SparklesIcon, TrashIcon,} from '@heroicons/vue/24/outline'
import {useUIStore} from "@/stores/ui.ts";

const uiStore = useUIStore();

// Tab 配置
const tabs = [
  { id: 'security', name: '安全设置', icon: ShieldCheckIcon },
  { id: 'storage', name: '存储设置', icon: CloudIcon },
  { id: 'cleanup', name: '清理策略', icon: TrashIcon },
  { id: 'ai', name: 'AI 设置', icon: SparklesIcon },
]
</script>
