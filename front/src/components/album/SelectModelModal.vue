<template>
  <Modal v-model="isOpen" size="sm" title="选择向量模型">
    <div class="space-y-4">
      <!-- 说明文字 -->
      <div class="text-sm text-gray-400">
        选择一个向量模型来计算相册的平均向量封面。系统将自动选择最接近平均向量的图片作为封面。
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <!-- 模型列表 -->
      <div v-else-if="models.length > 0" class="space-y-2">
        <button
          v-for="model in models"
          :key="model"
          :class="[
            selectedModel === model
              ? 'bg-primary-500/20 ring-1 ring-primary-500/50'
              : 'bg-white/5 hover:bg-white/10'
          ]"
          class="w-full flex items-center justify-between rounded-xl px-4 py-3 transition-all text-left"
          @click="selectedModel = model"
        >
          <span class="text-sm text-white font-medium">{{ model }}</span>
          <div
            :class="[
              selectedModel === model
                ? 'border-primary-500 bg-primary-500 text-white'
                : 'border-white/30'
            ]"
            class="flex h-5 w-5 items-center justify-center rounded-full border-2 transition-colors flex-shrink-0"
          >
            <CheckIcon v-if="selectedModel === model" class="h-3 w-3" />
          </div>
        </button>
      </div>

      <!-- 空状态 -->
      <div v-else class="py-8 text-center">
        <p class="text-sm text-gray-400">未找到可用的向量模型</p>
      </div>

      <!-- 操作按钮 -->
      <div class="flex justify-end gap-3 pt-2">
        <button
          class="px-5 py-2.5 rounded-xl border border-white/10 text-gray-400 hover:bg-white/5 transition-colors"
          type="button"
          @click="isOpen = false"
        >
          取消
        </button>
        <button
          :disabled="!selectedModel || submitting"
          class="px-5 py-2.5 rounded-xl bg-primary-500 text-white hover:bg-primary-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          @click="handleConfirm"
        >
          {{ submitting ? '设置中...' : '确定' }}
        </button>
      </div>
    </div>
  </Modal>
</template>

<script lang="ts" setup>
import {ref, watch} from 'vue'
import Modal from '@/components/common/Modal.vue'
import {CheckIcon} from '@heroicons/vue/24/outline'
import {aiApi} from '@/api/ai'
import {useDialogStore} from "@/stores/dialog.ts";

const dialogStore = useDialogStore();

const isOpen = defineModel<boolean>({ default: false })
const emit = defineEmits<{
  selected: [modelName: string]
}>()

const models = ref<string[]>([])
const selectedModel = ref<string>('')
const loading = ref(false)
const submitting = ref(false)

// 加载可用的模型列表
async function loadModels() {
  try {
    loading.value = true
    // 从 AI API 获取可用的嵌入模型列表
    const { data } = await aiApi.getEmbeddingModels()

    models.value = data || []

    // 默认选中第一个模型
    if (models.value.length > 0 && !selectedModel.value) {
      selectedModel.value = models.value[0]!
    }
  } catch (err) {
    dialogStore.notify({
      title: '加载模型列表失败',
      message: (err as Error).message,
      type: 'error'
    })
  } finally {
    loading.value = false
  }
}

function handleConfirm() {
  if (!selectedModel.value) return
  submitting.value = true
  emit('selected', selectedModel.value)
  // 延迟关闭，让父组件有时间处理
  setTimeout(() => {
    submitting.value = false
    isOpen.value = false
  }, 100)
}

// 监听弹窗打开/关闭
watch(isOpen, (val) => {
  if (val) {
    loadModels()
  } else {
    selectedModel.value = ''
  }
})
</script>
