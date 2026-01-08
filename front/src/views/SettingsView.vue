<template>
  <AppLayout>
    <template #header>
      <header
          class="sticky top-0 z-30 flex h-16 w-full items-center justify-between border-b border-white/5 bg-black/20 backdrop-blur-xl px-6 transition-all duration-300">
        <h1 class="text-xl font-bold tracking-wide text-white/90 font-display">系统设置</h1>
        <div v-if="hasUnsavedChanges" class="flex items-center gap-3">
          <button
              :disabled="saving"
              class="px-6 py-2.5 rounded-xl border border-white/10 bg-white/5  text-sm font-medium text-white hover:bg-white/10 transition-colors"
              @click="handleRestore"
          >
            还原配置
          </button>
          <button
              :disabled="saving"
              class="px-6 py-2.5 rounded-xl bg-primary-500/20  text-sm  hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              @click="handleSave"
          >
            <span v-if="saving"
                  class="h-4 w-4 animate-spin rounded-full border-2 border-primary-400 border-t-transparent"></span>
            {{ saving ? '保存中...' : '保存设置' }}
          </button>
        </div>
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
                @click="handleTabClick(tab.id)"
            >
              <component :is="tab.icon" class="h-4 w-4 inline-block mr-2"/>
              {{ tab.name }}
            </button>
          </div>

          <!-- 安全设置 Tab -->
          <div v-if="uiStore.settingActiveTab === 'security'" class="space-y-6">
            <SecuritySettings ref="securityRef" @change="handleChange" @saving="handleSavingChange"/>
          </div>

          <!-- 存储设置 Tab -->
          <div v-if="uiStore.settingActiveTab === 'storage'" class="space-y-6">
            <StorageSettings ref="storageRef" @change="handleChange" @saving="handleSavingChange"/>
          </div>

          <!-- 清理策略 Tab -->
          <div v-if="uiStore.settingActiveTab === 'cleanup'" class="space-y-6">
            <CleanupSettings ref="cleanupRef" @change="handleChange" @saving="handleSavingChange"/>
          </div>

          <!-- AI 设置 Tab -->
          <div v-if="uiStore.settingActiveTab === 'ai'" class="space-y-6">
            <AISettings ref="aiRef" @change="handleChange" @saving="handleSavingChange"/>
          </div>
        </div>
      </div>
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {onBeforeRouteLeave} from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import SecuritySettings from '@/components/settings/SecuritySettings.vue'
import StorageSettings from '@/components/settings/StorageSettings.vue'
import CleanupSettings from '@/components/settings/CleanupSettings.vue'
import AISettings from '@/components/settings/AISettings.vue'
import {CloudIcon, ShieldCheckIcon, SparklesIcon, TrashIcon,} from '@heroicons/vue/24/outline'
import {useUIStore} from "@/stores/ui.ts"
import {useDialogStore} from "@/stores/dialog.ts"

const uiStore = useUIStore()
const dialogStore = useDialogStore()

// Tab 配置
const tabs = [
  {id: 'security', name: '安全设置', icon: ShieldCheckIcon},
  {id: 'storage', name: '存储设置', icon: CloudIcon},
  {id: 'cleanup', name: '清理策略', icon: TrashIcon},
  {id: 'ai', name: 'AI 设置', icon: SparklesIcon},
]

// 各设置组件的引用
const securityRef = ref<InstanceType<typeof SecuritySettings> | null>(null)
const storageRef = ref<InstanceType<typeof StorageSettings> | null>(null)
const cleanupRef = ref<InstanceType<typeof CleanupSettings> | null>(null)
const aiRef = ref<InstanceType<typeof AISettings> | null>(null)

// 追踪是否有未保存的更改
const hasUnsavedChanges = ref(false)
const saving = ref(false)

// 获取当前激活的设置组件引用
const currentSettingRef = computed(() => {
  switch (uiStore.settingActiveTab) {
    case 'security':
      return securityRef.value
    case 'storage':
      return storageRef.value
    case 'cleanup':
      return cleanupRef.value
    case 'ai':
      return aiRef.value
    default:
      return null
  }
})

// 处理设置项变更
function handleChange(hasChanges: boolean) {
  hasUnsavedChanges.value = hasChanges
}

// 处理保存状态变更
function handleSavingChange(isSaving: boolean) {
  saving.value = isSaving
}

// 统一保存方法
async function handleSave() {
  const settingComponent = currentSettingRef.value
  if (settingComponent && typeof settingComponent.save === 'function') {
    await settingComponent.save()
  }
}

// 还原配置方法
async function handleRestore() {
  const result = await dialogStore.confirm({
    title: '确认还原',
    message: '确定要还原到上次保存的配置吗？当前的修改将会丢失。',
    type: 'warning',
    confirmText: '确认还原',
    cancelText: '取消'
  })

  if (result) {
    const settingComponent = currentSettingRef.value
    if (settingComponent && typeof settingComponent.restore === 'function') {
      settingComponent.restore()
    }
    hasUnsavedChanges.value = false
  }
}


// 处理tab点击
async function handleTabClick(tabId: string) {
  if (hasUnsavedChanges.value) {
    // 有未保存的更改，询问用户
    const result = await dialogStore.confirm({
      title: '未保存的更改',
      message: '您有未保存的更改，是否要保存？',
      type: 'warning',
      confirmText: '保存',
      cancelText: '取消'
    })
    if (!result) return
    await handleSave()
    // 切换tab后重置未保存状态
    hasUnsavedChanges.value = false
  }

  uiStore.settingActiveTab = tabId
}


// 离开页面前检查
onBeforeRouteLeave(async (_to, _from, next) => {
  if (hasUnsavedChanges.value) {
    const result = await dialogStore.confirm({
      title: '未保存的更改',
      message: '您有未保存的更改，是否要保存？',
      type: 'warning',
      confirmText: '保存',
      cancelText: '取消'
    })

    if (result) {
      await handleSave()
      next()
    } else {
      // 用户选择取消，阻止离开
      next(false)
    }
  } else {
    next()
  }
})
</script>
