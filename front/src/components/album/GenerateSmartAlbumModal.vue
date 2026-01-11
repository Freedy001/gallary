<template>
  <Modal v-model="isOpen" size="md" title="生成智能相册">
    <form class="space-y-6" @submit.prevent="handleSubmit">
      <!-- 模型选择 -->
      <div>
        <BaseSelect
            v-model="form.model_name"
            :disabled="smartAlbumStore.taskInProgress"
            :options="embeddingModelOptions"
            label="嵌入模型"
            placeholder="请选择模型"
        />
        <p class="text-xs text-gray-500 mt-1">选择用于获取图片向量的嵌入模型</p>
      </div>

      <!-- 算法选择 -->
      <div>
        <BaseSelect
            v-model="form.algorithm"
            :disabled="smartAlbumStore.taskInProgress"
            :options="algorithmOptions"
            label="聚类算法"
        />
      </div>

      <!-- 高级配置折叠 -->
      <div class="border border-white/10 rounded-xl overflow-hidden">
        <button
            :disabled="smartAlbumStore.taskInProgress"
            class="w-full flex items-center justify-between px-4 py-3 text-sm text-gray-300 hover:bg-white/5 transition-colors disabled:opacity-50"
            type="button"
            @click="showAdvanced = !showAdvanced"
        >
          <span>高级配置</span>
          <ChevronDownIcon :class="['h-4 w-4 transition-transform', showAdvanced ? 'rotate-180' : '']"/>
        </button>

        <div v-show="showAdvanced" class="px-4 pb-4 space-y-4 border-t border-white/10 pt-4">
          <!-- HDBSCAN 参数 -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-xs text-gray-400 mb-1">最小聚类大小</label>
              <div class="relative">
                <input
                    v-model.number="form.hdbscan_params.min_cluster_size"
                    :disabled="smartAlbumStore.taskInProgress"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-sm focus:border-primary-500 outline-none disabled:opacity-50 appearance-none [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                    min="2"
                    type="number"
                />
              </div>
              <p class="text-[10px] text-gray-500 mt-0.5">每个相册最少包含的图片数</p>
            </div>
            <div>
              <label class="block text-xs text-gray-400 mb-1">抗噪能力</label>
              <div class="relative">
                <input
                    v-model.number="form.hdbscan_params.min_samples"
                    :disabled="smartAlbumStore.taskInProgress"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-sm focus:border-primary-500 outline-none placeholder-gray-600 disabled:opacity-50 appearance-none [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                    min="1"
                    placeholder="默认等于最小聚类大小"
                    type="number"
                />
              </div>
              <p class="text-[10px] text-gray-500 mt-0.5">越小保留更多图片，越大聚类更纯净</p>
            </div>
            <div>
              <label class="block text-xs text-gray-400 mb-1">聚类选择方法</label>
              <BaseSelect
                  v-model="form.hdbscan_params.cluster_selection_method"
                  :disabled="smartAlbumStore.taskInProgress"
                  :options="clusterSelectionMethodOptions"
                  button-class="!py-2 !text-sm"
              />
              <p class="text-[10px] text-gray-500 mt-0.5">EOM 更稳定，Leaf 产生更多小聚类</p>
            </div>
            <div>
              <label class="block text-xs text-gray-400 mb-1">聚类合并阈值</label>
              <div class="relative">
                <input
                    v-model.number="form.hdbscan_params.cluster_selection_epsilon"
                    :disabled="smartAlbumStore.taskInProgress"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-sm focus:border-primary-500 outline-none disabled:opacity-50 appearance-none [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                    min="0"
                    step="0.1"
                    type="number"
                />
              </div>
              <p class="text-[10px] text-gray-500 mt-0.5">合并相近的小聚类，0 表示不合并</p>
            </div>
          </div>

          <!-- UMAP 降维 -->
          <div class="pt-2 border-t border-white/5">
            <div class="flex items-center justify-between">
              <div>
                <span class="text-sm text-gray-300">启用 UMAP 降维</span>
                <p class="text-[10px] text-gray-500 mt-0.5">图片数量较多时建议启用，可提高聚类效果</p>
              </div>
              <button
                  :aria-checked="form.hdbscan_params.umap_enabled"
                  :class="[
                  form.hdbscan_params.umap_enabled ? 'bg-primary-500' : 'bg-white/10',
                  smartAlbumStore.taskInProgress ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
                  'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none'
                ]"
                  :disabled="smartAlbumStore.taskInProgress"
                  role="switch"
                  type="button"
                  @click="!smartAlbumStore.taskInProgress && (form.hdbscan_params.umap_enabled = !form.hdbscan_params.umap_enabled)"
              >
                <span
                    :class="[
                    form.hdbscan_params.umap_enabled ? 'translate-x-5' : 'translate-x-0',
                    'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out'
                  ]"
                    aria-hidden="true"
                />
              </button>
            </div>

            <!-- UMAP 参数（仅在启用时显示） -->
            <div v-show="form.hdbscan_params.umap_enabled" class="mt-4 grid grid-cols-2 gap-4">
              <div>
                <label class="block text-xs text-gray-400 mb-1">邻居数量</label>
                <div class="relative">
                  <input
                      v-model.number="form.hdbscan_params.umap_n_neighbors"
                      :disabled="smartAlbumStore.taskInProgress"
                      class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-sm focus:border-primary-500 outline-none disabled:opacity-50 appearance-none [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                      max="200"
                      min="2"
                      type="number"
                  />
                </div>
                <p class="text-[10px] text-gray-500 mt-0.5">小值关注细节，大值关注语义</p>
              </div>
              <div>
                <label class="block text-xs text-gray-400 mb-1">最小距离</label>
                <div class="relative">
                  <input
                      v-model.number="form.hdbscan_params.umap_min_dist"
                      :disabled="smartAlbumStore.taskInProgress"
                      class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-sm focus:border-primary-500 outline-none disabled:opacity-50 appearance-none [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                      max="1"
                      min="0"
                      step="0.05"
                      type="number"
                  />
                </div>
                <p class="text-[10px] text-gray-500 mt-0.5">值越小聚类越紧凑</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="flex justify-end gap-3 pt-4">
        <button
            class="px-5 py-2.5 rounded-xl border border-white/10 text-gray-400 hover:bg-white/5 hover:text-white transition-colors"
            type="button"
            @click="handleClose"
        >
          取消
        </button>
        <button
            :disabled="loading || !form.model_name || smartAlbumStore.taskInProgress"
            class="px-5 py-2.5 rounded-xl bg-primary-500 text-white hover:bg-primary-600 shadow-glow hover:shadow-[0_0_20px_rgba(139,92,246,0.5)] transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none"
            type="submit"
        >
          {{ loading ? '提交中...' : '生成智能相册' }}
        </button>
      </div>
    </form>
  </Modal>
</template>

<script lang="ts" setup>
import {computed, onMounted, reactive, ref, watch} from 'vue'
import {ChevronDownIcon} from '@heroicons/vue/24/outline'
import Modal from '@/components/common/Modal.vue'
import type {SelectOption} from '@/components/common/BaseSelect.vue'
import BaseSelect from '@/components/common/BaseSelect.vue'
import {aiApi} from '@/api/ai'
import type {EmbeddingModelInfo} from '@/types/ai'
import type {GenerateSmartAlbumsRequest} from '@/types/smart-album'
import {DEFAULT_HDBSCAN_PARAMS} from '@/types/smart-album'
import {useDialogStore} from '@/stores/dialog'
import {useSmartAlbumStore} from '@/stores/smartAlbum'

const isOpen = defineModel<boolean>({default: false})
const emit = defineEmits<{
  generated: []
}>()

const dialogStore = useDialogStore()
const smartAlbumStore = useSmartAlbumStore()

const loading = ref(false)
const showAdvanced = ref(false)
const embeddingModels = ref<EmbeddingModelInfo[]>([])

// 计算选项列表（使用模型名称作为显示和值）
const embeddingModelOptions = computed<SelectOption[]>(() =>
    embeddingModels.value.map(model => ({label: model.model_name, value: model.model_name}))
)

const algorithmOptions: SelectOption[] = [
  {label: 'HDBSCAN（密度聚类）', value: 'hdbscan'}
]

const clusterSelectionMethodOptions: SelectOption[] = [
  {label: 'EOM（推荐）', value: 'eom'},
  {label: 'Leaf', value: 'leaf'}
]

const form = reactive<GenerateSmartAlbumsRequest>({
  model_name: '',
  algorithm: 'hdbscan',
  hdbscan_params: {...DEFAULT_HDBSCAN_PARAMS}
})

// 打开弹窗时重置状态
watch(isOpen, (val) => {
  if (val) {
    // 只有在没有进行中任务时才重置
    if (!smartAlbumStore.taskInProgress) {
      smartAlbumStore.resetState()
      form.hdbscan_params = {...DEFAULT_HDBSCAN_PARAMS}
    }
  }
})

// 监听结果，如果完成则触发事件
watch(() => smartAlbumStore.result, (newResult) => {
  if (newResult) {
    emit('generated')
  }
})

onMounted(async () => {
  try {
    const res = await aiApi.getEmbeddingModels()
    const models = res.data || []
    embeddingModels.value = models
    const firstModel = models[0]
    if (firstModel) {
      form.model_name = firstModel.model_name
    }
  } catch (err) {
    console.error('获取嵌入模型列表失败', err)
  }
})

async function handleSubmit() {
  if (!form.model_name || loading.value || smartAlbumStore.taskInProgress) return

  try {
    loading.value = true
    smartAlbumStore.resetState()

    const res = await aiApi.generateSmartAlbum(form)
    smartAlbumStore.setTaskId(res.data.task_id)

    dialogStore.notify({
      title: '任务已提交',
      message: '智能相册生成任务已开始，进度将实时更新',
      type: 'info'
    })

    // 提交成功后立即关闭弹窗
    isOpen.value = false
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
