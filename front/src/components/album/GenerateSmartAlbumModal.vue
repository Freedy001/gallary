<template>
  <Modal v-model="isOpen" size="md" title="生成智能相册">
    <form class="space-y-6" @submit.prevent="handleSubmit">
      <!-- 模型选择 -->
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">嵌入模型</label>
        <select
          v-model="form.model_name"
          :disabled="taskInProgress"
          class="w-full rounded-xl bg-white/5 border border-white/10 px-4 py-3 text-white focus:border-primary-500 focus:ring-1 focus:ring-primary-500 transition-colors outline-none disabled:opacity-50"
          required
        >
          <option disabled value="">请选择模型</option>
          <option v-for="model in embeddingModels" :key="model" :value="model">
            {{ model }}
          </option>
        </select>
        <p class="text-xs text-gray-500 mt-1">选择用于获取图片向量的嵌入模型</p>
      </div>

      <!-- 算法选择 -->
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">聚类算法</label>
        <select
          v-model="form.algorithm"
          :disabled="taskInProgress"
          class="w-full rounded-xl bg-white/5 border border-white/10 px-4 py-3 text-white focus:border-primary-500 focus:ring-1 focus:ring-primary-500 transition-colors outline-none disabled:opacity-50"
        >
          <option value="hdbscan">HDBSCAN（密度聚类）</option>
        </select>
      </div>

      <!-- 高级配置折叠 -->
      <div class="border border-white/10 rounded-xl overflow-hidden">
        <button
          :disabled="taskInProgress"
          class="w-full flex items-center justify-between px-4 py-3 text-sm text-gray-300 hover:bg-white/5 transition-colors disabled:opacity-50"
          type="button"
          @click="showAdvanced = !showAdvanced"
        >
          <span>高级配置</span>
          <ChevronDownIcon :class="['h-4 w-4 transition-transform', showAdvanced ? 'rotate-180' : '']" />
        </button>

        <div v-show="showAdvanced" class="px-4 pb-4 space-y-4 border-t border-white/10 pt-4">
          <!-- HDBSCAN 参数 -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-xs text-gray-400 mb-1">最小聚类大小</label>
              <input
                v-model.number="form.hdbscan_params.min_cluster_size"
                :disabled="taskInProgress"
                class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-sm focus:border-primary-500 outline-none disabled:opacity-50"
                min="2"
                type="number"
              />
              <p class="text-[10px] text-gray-500 mt-0.5">每个相册最少包含的图片数</p>
            </div>
            <div>
              <label class="block text-xs text-gray-400 mb-1">最小样本数</label>
              <input
                v-model.number="form.hdbscan_params.min_samples"
                :disabled="taskInProgress"
                class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-sm focus:border-primary-500 outline-none placeholder-gray-600 disabled:opacity-50"
                min="1"
                placeholder="默认等于最小聚类大小"
                type="number"
              />
              <p class="text-[10px] text-gray-500 mt-0.5">核心点判定标准</p>
            </div>
            <div>
              <label class="block text-xs text-gray-400 mb-1">聚类选择方法</label>
              <select
                v-model="form.hdbscan_params.cluster_selection_method"
                :disabled="taskInProgress"
                class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-sm focus:border-primary-500 outline-none disabled:opacity-50"
              >
                <option value="eom">EOM（推荐）</option>
                <option value="leaf">Leaf</option>
              </select>
              <p class="text-[10px] text-gray-500 mt-0.5">EOM 更稳定，Leaf 产生更多小聚类</p>
            </div>
            <div>
              <label class="block text-xs text-gray-400 mb-1">距离度量</label>
              <select
                v-model="form.hdbscan_params.metric"
                :disabled="taskInProgress"
                class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-sm focus:border-primary-500 outline-none disabled:opacity-50"
              >
                <option value="cosine">余弦相似度（推荐）</option>
                <option value="euclidean">欧氏距离</option>
              </select>
            </div>
          </div>

          <!-- UMAP 降维 -->
          <div class="pt-2 border-t border-white/5">
            <label class="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
              <input
                v-model="form.hdbscan_params.umap_enabled"
                :disabled="taskInProgress"
                class="rounded bg-white/5 border-white/10 text-primary-500 focus:ring-primary-500 disabled:opacity-50"
                type="checkbox"
              />
              启用 UMAP 降维
            </label>
            <p class="text-xs text-gray-500 mt-1 ml-6">图片数量较多时建议启用，可提高聚类效果</p>
          </div>
        </div>
      </div>

      <!-- 任务进度 -->
      <div v-if="currentProgress" class="p-4 bg-white/5 border border-white/10 rounded-xl space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-300">{{ getStatusText(currentProgress.status) }}</span>
          <span class="text-xs text-gray-500">{{ currentProgress.progress }}%</span>
        </div>
        <div class="w-full bg-white/10 rounded-full h-2 overflow-hidden">
          <div
            :class="getProgressBarClass(currentProgress.status)"
            :style="{ width: `${currentProgress.progress}%` }"
            class="h-full rounded-full transition-all duration-300"
          />
        </div>
        <p class="text-xs text-gray-400">{{ currentProgress.message }}</p>
      </div>

      <!-- 操作按钮 -->
      <div class="flex justify-end gap-3 pt-4">
        <button
          class="px-5 py-2.5 rounded-xl border border-white/10 text-gray-400 hover:bg-white/5 transition-colors"
          type="button"
          @click="handleClose"
        >
          {{ taskInProgress ? '后台运行' : '取消' }}
        </button>
        <button
          :disabled="loading || !form.model_name || taskInProgress"
          class="px-5 py-2.5 rounded-xl bg-gradient-to-r from-purple-500 to-blue-500 text-white hover:from-purple-600 hover:to-blue-600 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          type="submit"
        >
          {{ loading ? '提交中...' : '生成智能相册' }}
        </button>
      </div>
    </form>

    <!-- 生成结果 -->
    <div v-if="result" class="mt-6 p-4 bg-green-500/10 border border-green-500/20 rounded-xl">
      <div class="flex items-center gap-2 text-green-400 mb-2">
        <CheckCircleIcon class="h-5 w-5" />
        <span class="font-medium">生成完成</span>
      </div>
      <div class="text-sm text-gray-300 space-y-1">
        <p>已创建 <span class="text-white font-medium">{{ result.cluster_count }}</span> 个智能相册</p>
        <p>共处理 <span class="text-white font-medium">{{ result.total_images }}</span> 张图片</p>
        <p v-if="result.noise_count && result.noise_count > 0" class="text-gray-400">
          {{ result.noise_count }} 张图片未被归类（噪声点）
        </p>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="errorMessage" class="mt-6 p-4 bg-red-500/10 border border-red-500/20 rounded-xl">
      <div class="flex items-center gap-2 text-red-400 mb-2">
        <XCircleIcon class="h-5 w-5" />
        <span class="font-medium">生成失败</span>
      </div>
      <p class="text-sm text-gray-300">{{ errorMessage }}</p>
    </div>
  </Modal>
</template>

<script lang="ts" setup>
import {computed, onMounted, onUnmounted, reactive, ref, watch} from 'vue'
import {CheckCircleIcon, ChevronDownIcon, XCircleIcon} from '@heroicons/vue/24/outline'
import Modal from '@/components/common/Modal.vue'
import {smartAlbumApi} from '@/api/smart-album'
import {aiApi} from '@/api/ai'
import type {GenerateSmartAlbumsRequest, SmartAlbumTaskStatus} from '@/types/smart-album'
import {DEFAULT_HDBSCAN_PARAMS} from '@/types/smart-album'
import {useDialogStore} from '@/stores/dialog'
import wsService from '@/services/websocket'

// 智能相册进度 VO（与后端一致）
interface SmartAlbumProgressVO {
  task_id: number
  status: SmartAlbumTaskStatus
  progress: number
  message: string
  error?: string
  album_ids?: number[]
  cluster_count?: number
  noise_count?: number
  total_images?: number
}

const isOpen = defineModel<boolean>({ default: false })
const emit = defineEmits<{
  generated: []
}>()

const dialogStore = useDialogStore()

const loading = ref(false)
const showAdvanced = ref(false)
const embeddingModels = ref<string[]>([])
const currentTaskId = ref<number | null>(null)
const currentProgress = ref<SmartAlbumProgressVO | null>(null)
const result = ref<SmartAlbumProgressVO | null>(null)
const errorMessage = ref<string | null>(null)
let unsubscribe: (() => void) | null = null

const form = reactive<GenerateSmartAlbumsRequest>({
  model_name: '',
  algorithm: 'hdbscan',
  hdbscan_params: { ...DEFAULT_HDBSCAN_PARAMS }
})

const taskInProgress = computed(() => {
  if (!currentProgress.value) return false
  const status = currentProgress.value.status
  return status === 'pending' || status === 'collecting' || status === 'clustering' || status === 'creating'
})

function getStatusText(status: SmartAlbumTaskStatus): string {
  const statusMap: Record<SmartAlbumTaskStatus, string> = {
    pending: '等待处理',
    collecting: '收集向量数据',
    clustering: '执行聚类分析',
    creating: '创建相册',
    completed: '已完成',
    failed: '失败'
  }
  return statusMap[status] || status
}

function getProgressBarClass(status: SmartAlbumTaskStatus): string {
  if (status === 'failed') return 'bg-red-500'
  if (status === 'completed') return 'bg-green-500'
  return 'bg-gradient-to-r from-purple-500 to-blue-500'
}

// 处理 WebSocket 进度消息
function handleProgressMessage(data: SmartAlbumProgressVO) {
  // 只处理当前任务的进度
  if (currentTaskId.value && data.task_id === currentTaskId.value) {
    currentProgress.value = data

    if (data.status === 'completed') {
      result.value = data
      dialogStore.notify({
        title: '成功',
        message: `成功创建 ${data.cluster_count || 0} 个智能相册`,
        type: 'success'
      })
      emit('generated')
    } else if (data.status === 'failed') {
      errorMessage.value = data.error || '任务执行失败'
    }
  }
}

// 打开弹窗时重置状态
watch(isOpen, (val) => {
  if (val) {
    result.value = null
    errorMessage.value = null
    currentProgress.value = null
    currentTaskId.value = null
    form.hdbscan_params = { ...DEFAULT_HDBSCAN_PARAMS }

    // 订阅 WebSocket 消息
    unsubscribe = wsService.subscribe<SmartAlbumProgressVO>('smart_album_progress', handleProgressMessage)
  } else {
    // 取消订阅
    if (unsubscribe) {
      unsubscribe()
      unsubscribe = null
    }
  }
})

onMounted(async () => {
  try {
    const res = await aiApi.getEmbeddingModels()
    const models = res.data || []
    embeddingModels.value = models
    const firstModel = models[0]
    if (firstModel) {
      form.model_name = firstModel
    }
  } catch (err) {
    console.error('获取嵌入模型列表失败', err)
  }
})

onUnmounted(() => {
  if (unsubscribe) {
    unsubscribe()
    unsubscribe = null
  }
})

async function handleSubmit() {
  if (!form.model_name || loading.value || taskInProgress.value) return

  try {
    loading.value = true
    result.value = null
    errorMessage.value = null

    const res = await smartAlbumApi.submitTask(form)
    const taskVO = res.data as unknown as SmartAlbumProgressVO
    currentTaskId.value = taskVO.task_id
    currentProgress.value = taskVO

    dialogStore.notify({
      title: '任务已提交',
      message: '智能相册生成任务已开始，进度将实时更新',
      type: 'info'
    })
  } catch (err: any) {
    console.error('提交任务失败', err)
    dialogStore.notify({
      title: '错误',
      message: err.response?.data?.message || '提交失败，请重试',
      type: 'error'
    })
  } finally {
    loading.value = false
  }
}

function handleClose() {
  isOpen.value = false
}
</script>
