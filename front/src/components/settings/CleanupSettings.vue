<template>
  <div class="rounded-2xl bg-white/5 ring-1 ring-white/10 overflow-hidden">
    <div class="border-b border-white/5 p-5 bg-white/2">
      <h2 class="text-lg font-medium text-white">自动清理策略</h2>
      <p class="mt-1 text-sm text-gray-500">配置回收站图片的自动清理规则</p>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="p-6 flex justify-center">
      <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
    </div>

    <div v-else class="p-6 space-y-6">
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-3">回收站自动删除天数</label>
        <div class="flex items-center gap-4">
          <input
              v-model.number="days"
              type="range"
              min="0"
              max="365"
              class="flex-1 h-2 bg-white/10 rounded-lg appearance-none cursor-pointer accent-primary-500"
          />
          <div class="w-24 text-center">
            <input
                v-model.number="days"
                type="number"
                min="0"
                max="365"
                class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-center focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
            />
          </div>
          <span class="text-gray-400 text-sm">天</span>
        </div>
        <p class="mt-3 text-xs text-gray-500">
          {{ days === 0 ? '已禁用自动清理' : `回收站中的图片将在 ${days} 天后自动永久删除` }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {onMounted, ref, watch} from 'vue'
import {type CleanupConfig, settingsApi} from '@/api/settings'
import {useDialogStore} from '@/stores/dialog'

const dialogStore = useDialogStore()

// 定义 emits
const emit = defineEmits<{
  change: [hasChanges: boolean]
  saving: [isSaving: boolean]
}>()

const loading = ref(true)
const saving = ref(false)
const days = ref(30)
const originalDays = ref(30)

async function loadSettings() {
  loading.value = true
  try {
    const resp = await settingsApi.getByCategory('cleanup')
    const data = resp.data
    if (data.trash_auto_delete_days !== undefined) {
      days.value = data.trash_auto_delete_days
      originalDays.value = data.trash_auto_delete_days
    }
  } catch (error) {
    console.error('Failed to load cleanup settings:', error)
  } finally {
    loading.value = false
  }
}

// 监听days变化
watch(days, (newValue) => {
  emit('change', newValue !== originalDays.value)
})

async function handleSave() {
  saving.value = true
  emit('saving', true)
  try {
    const config: CleanupConfig = {
      trash_auto_delete_days: days.value
    }
    await settingsApi.updateCleanup(config)
    originalDays.value = days.value
    emit('change', false)
    dialogStore.alert({ title: '成功', message: '清理策略更新成功', type: 'success' })
  } catch (error: any) {
    dialogStore.alert({ title: '错误', message: error.message || '更新清理策略失败', type: 'error' })
  } finally {
    saving.value = false
    emit('saving', false)
  }
}

// 暴露 save 方法
function save() {
  return handleSave()
}

// 还原配置方法
function restore() {
  days.value = originalDays.value
  emit('change', false)
}

defineExpose({
  save,
  restore
})

onMounted(() => {
  loadSettings()
})
</script>
