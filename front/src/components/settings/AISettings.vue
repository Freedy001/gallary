<template>
  <div class="space-y-6">
    <!-- AI 全局设置 -->
    <div class="rounded-2xl bg-white/5 ring-1 ring-white/10">
      <div class="border-b border-white/5 p-5 bg-white/2 rounded-t-2xl">
        <h2 class="text-lg font-medium text-white">全局设置</h2>
        <p class="mt-1 text-sm text-gray-500">配置 AI 功能的默认行为</p>
      </div>

      <div class="p-6 space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <!-- 默认搜索模型 -->
          <div>
            <BaseSelect
                v-model="config.global_config!.default_search_model_id"
                :options="embeddingModelOptions"
                label="默认搜索模型"
                placeholder="选择默认搜索模型"
            />
            <p class="mt-1 text-xs text-gray-500">用于语义搜索的默认模型</p>
          </div>

          <!-- 默认打标签模型 -->
          <div>
            <BaseSelect
                v-model="config.global_config!.default_tag_model_id"
                :options="embeddingModelOptions"
                label="默认打标签模型"
                placeholder="选择默认打标签模型"
            />
            <p class="mt-1 text-xs text-gray-500">用于自动生成图片标签的默认模型</p>
          </div>

          <!-- 默认提示词优化模型 -->
          <div>
            <BaseSelect
                v-model="config.global_config!.default_prompt_optimize_model_id"
                :options="llmsModelOptions"
                label="默认提示词优化模型"
                placeholder="选择默认提示词优化模型"
            />
            <p class="mt-1 text-xs text-gray-500">用于搜索提示词优化默认模型</p>
          </div>

          <!-- 默认命名模型 -->
          <div>
            <BaseSelect
                v-model="config.global_config!.default_naming_model_id"
                :options="llmsModelOptions"
                label="默认命名模型"
                placeholder="选择默认命名模型"
            />
            <p class="mt-1 text-xs text-gray-500">用于 AI 智能命名相册的默认模型</p>
          </div>
        </div>

        <!-- 提示词优化配置 -->
        <div class="pt-4 border-t border-white/5">
          <div class="flex items-center justify-between mb-3">
            <div>
              <h4 class="text-sm font-medium text-white">搜索提示词优化</h4>
            </div>
          </div>

          <!-- 系统提示词输入框 -->
          <div>
            <textarea
                v-model="config.global_config!.prompt_optimize_system_prompt"
                class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm resize-y"
                placeholder="留空使用默认提示词..."
                rows="4"
            />
            <p class="mt-1 text-xs text-gray-500">默认提示词会将中文搜索词翻译为英文并扩展为详细的视觉描述</p>
          </div>
        </div>

        <!-- 命名提示词配置 -->
        <div class="pt-4 border-t border-white/5">
          <div class="flex items-center justify-between mb-3">
            <div>
              <h4 class="text-sm font-medium text-white">相册命名配置</h4>
            </div>
          </div>

          <!-- 最大图片数量 -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-400 mb-1.5">命名使用的最大图片数量</label>
            <input
                v-model.number="config.global_config!.naming_max_images"
                class="w-32 rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none [-moz-appearance:textfield]"
                max="10"
                min="1"
                placeholder="默认: 3"
                type="number"
            />
            <p class="mt-1 text-xs text-gray-500">AI 命名时从相册中选取的代表性图片数量（默认 3）</p>
          </div>

          <!-- 命名提示词输入框 -->
          <div>
            <label class="block text-sm font-medium text-gray-400 mb-1.5">命名系统提示词</label>
            <textarea
                v-model="config.global_config!.naming_system_prompt"
                class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm resize-y"
                placeholder="留空使用默认提示词..."
                rows="4"
            />
            <p class="mt-1 text-xs text-gray-500">用于根据相册内容生成合适的中文名称，留空使用默认提示词</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 嵌入模型设置 -->
    <div class="rounded-2xl bg-white/5 ring-1 ring-white/10 overflow-hidden">
      <div class="border-b border-white/5 p-5 bg-white/2 flex items-center justify-between">
        <div>
          <h2 class="text-lg font-medium text-white">模型配置</h2>
          <p class="mt-1 text-sm text-gray-500">配置用于图片向量嵌入和美学评分的模型（OpenAI 兼容格式），支持多模型配置</p>
        </div>
        <button
            @click="addProvider"
            class="px-3 py-1.5 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 text-sm flex items-center gap-1"
        >
          <PlusIcon class="h-4 w-4"/>
          添加提供商
        </button>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="p-6 flex justify-center">
        <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <!-- 无模型状态 -->
      <div v-else-if="config.models.length === 0" class="p-8 text-center">
        <div class="text-gray-500 mb-4">
          <svg class="h-12 w-12 mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                  d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/>
          </svg>
          暂无模型配置
        </div>
        <button
            @click="addProvider"
            class="px-4 py-2 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 text-sm"
        >
          添加第一个提供商
        </button>
      </div>

      <div v-else class="divide-y divide-white/5">
        <div v-for="(provider, index) in config.models" :key="index" class="p-6">
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-3">
              <!-- 启用开关 -->
              <button
                  :class="[
                    'relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                    provider.enabled ? 'bg-primary-500' : 'bg-gray-600'
                  ]"
                  @click="provider.enabled = !provider.enabled"
              >
                <span
                    :class="[
                      'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                      provider.enabled ? 'translate-x-4' : 'translate-x-0'
                    ]"
                />
              </button>
              <span class="text-white font-medium">{{ provider.id || '未命名提供商' }}</span>
              <span v-if="provider.provider === 'selfHosted'"
                    class="px-2 py-0.5 rounded text-xs bg-amber-500/20 text-amber-400">自托管</span>
            </div>
            <div class="flex items-center gap-2">
              <!-- 测试连接按钮 -->
              <button
                  :disabled="testingProvider === provider.id"
                  class="px-2 py-1 rounded text-xs text-gray-400 hover:text-primary-400 hover:bg-white/5 transition-colors flex items-center gap-1 disabled:opacity-50 disabled:cursor-not-allowed"
                  @click="testProviderConnection(provider)"
              >
                <span v-if="testingProvider === provider.id" class="h-3 w-3 animate-spin rounded-full border border-primary-400 border-t-transparent"></span>
                <span v-else>测试连接</span>
              </button>
              <button
                  @click="removeProvider(index)"
                  class="p-1 rounded text-gray-400 hover:text-red-400 hover:bg-white/5 transition-colors"
              >
                <TrashIcon class="h-4 w-4"/>
              </button>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <BaseSelect
                  v-model="provider.provider"
                  :options="providerOptions"
                  label="提供商类型"
                  @update:model-value="onProviderChange(provider)"
              />
            </div>
            <div v-if="provider.provider !== 'selfHosted'">
              <label class="block text-sm font-medium text-gray-400 mb-1.5">提供商 ID</label>
              <input
                  v-model="provider.id"
                  type="text"
                  placeholder="用于区分不同提供商配置"
                  class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-400 mb-1.5">API 端点</label>
              <input
                  v-model="provider.endpoint"
                  type="text"
                  placeholder="如: http://localhost:8100"
                  class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-400 mb-1.5">API Key</label>
              <input
                  v-model="provider.api_key"
                  type="password"
                  placeholder="留空则无需认证"
                  class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm"
              />
            </div>

            <!-- 模型列表（非自托管模型） -->
            <div v-if="provider.provider !== 'selfHosted'" class="col-span-2">
              <label class="block text-sm font-medium text-gray-400 mb-1.5">模型列表</label>
              <p class="text-xs text-gray-500 mb-2">同一提供商可配置多个模型，相同 model_name 的模型将被负载均衡</p>

              <!-- 已添加的模型标签 -->
              <div v-if="provider.models?.length" class="flex flex-wrap gap-2 mb-3">
                <button
                    v-for="(model, idx) in provider.models"
                    :key="idx"
                    class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-primary-500/20 text-primary-300 text-sm border border-primary-500/30 hover:bg-primary-500/30 hover:border-primary-500/50 transition-colors cursor-pointer"
                    @click="openEditModelDialog(provider, idx)"
                >
                  <span class="font-medium">{{ model.api_model_name }}</span>
                  <span v-if="model.model_name !== model.api_model_name"
                        class="text-gray-400 text-xs">({{ model.model_name }})</span>
                  <span
                      class="ml-1 hover:text-red-400 transition-colors"
                      @click.stop="removeModelFromProvider(provider, idx)"
                  >
                    <XMarkIcon class="h-3.5 w-3.5"/>
                  </span>
                </button>
              </div>

              <!-- 添加模型按钮 -->
              <button
                  class="mt-5 px-3 py-1.5 rounded-lg bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white border border-white/10 transition-colors text-sm flex items-center gap-1"
                  @click="openAddModelDialog(provider)"
              >
                <PlusIcon class="h-4 w-4"/>
                添加模型
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加/编辑模型对话框 -->
    <Modal v-model="showAddModelDialog" :title="isEditMode ? '编辑模型' : '添加模型'" size="sm">
      <form class="space-y-4" @submit.prevent="confirmAddModel">
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-2">API 模型名称 *</label>
          <input
              v-model="newModel.api_model_name"
              class="w-full rounded-xl bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 transition-colors outline-none"
              placeholder="调用 API 时使用的模型名称，如: gemini-2.5-flash"
              required
              type="text"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-300 mb-2">内部模型名称</label>
          <input
              v-model="newModel.model_name"
              :placeholder="newModel.api_model_name || '默认与 API 模型名称相同'"
              class="w-full rounded-xl bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 transition-colors outline-none"
              type="text"
          />
          <p class="mt-2 text-xs text-gray-500">相同名称的模型会被负载均衡，留空则与 API 模型名称相同</p>
        </div>

        <div class="flex justify-end gap-3 pt-4">
          <button
              class="px-5 py-2.5 rounded-xl border border-white/10 text-gray-400 hover:bg-white/5 transition-colors"
              type="button"
              @click="closeAddModelDialog"
          >
            取消
          </button>
          <button
              :disabled="!newModel.api_model_name"
              class="px-5 py-2.5 rounded-xl bg-primary-500 text-white hover:bg-primary-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              type="submit"
          >
            {{ isEditMode ? '保存' : '添加' }}
          </button>
        </div>
      </form>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import {computed, onMounted, reactive, ref, watch} from 'vue'
import {PlusIcon, TrashIcon, XMarkIcon} from '@heroicons/vue/24/outline'
import {useDialogStore} from '@/stores/dialog'
import type {AIConfig, ModelConfig, ModelItem} from '@/types/ai'
import {createModelId} from '@/types/ai'
import type {SelectOption} from '@/components/widgets/common/BaseSelect.vue'
import BaseSelect from '@/components/widgets/common/BaseSelect.vue'
import Modal from '@/components/widgets/common/Modal.vue'
import {aiApi} from "@/api/ai.ts"

const dialogStore = useDialogStore()

// 定义 emits
const emit = defineEmits<{
  change: [hasChanges: boolean]
  saving: [isSaving: boolean]
}>()

const loading = ref(true)
const saving = ref(false)

// 使用本地状态避免直接修改 store
const config = reactive<AIConfig>({
  models: [],
  global_config: {
    default_search_model_id: '',
    default_tag_model_id: '',
    default_prompt_optimize_model_id: '',
    prompt_optimize_system_prompt: '',
    default_naming_model_id: '',
    naming_system_prompt: '',
    naming_max_images: 3,
  }
})

// 原始配置，用于对比是否有变化
const originalConfig = reactive<AIConfig>({
  models: [],
  global_config: {
    default_search_model_id: '',
    default_tag_model_id: '',
    default_prompt_optimize_model_id: '',
    prompt_optimize_system_prompt: '',
    default_naming_model_id: '',
    naming_system_prompt: '',
    naming_max_images: 3,
  }
})

// 监听配置变化
watch(
  () => config,
  () => {
    const hasChanges = JSON.stringify(config) !== JSON.stringify(originalConfig)
    emit('change', hasChanges)
  },
  { deep: true }
)

// 计算启用的模型选项（用于全局设置下拉框）
// 生成所有 providerId,apiModelName 的组合
const embeddingModelOptions = computed<SelectOption[]>(() => {
  const options: SelectOption[] = []

  for (const provider of config.models) {
    if (!provider.enabled) continue
    if (provider.provider != 'openAI') {
      for (const model of (provider.models || [])) {
        const compositeId = createModelId(provider.id, model.api_model_name)
        options.push({
          label: `${model.api_model_name} (${provider.id})`,
          value: compositeId
        })
      }
    }

  }

  return options
})

const llmsModelOptions = computed<SelectOption[]>(() => {
  const options: SelectOption[] = []

  for (const provider of config.models) {
    if (!provider.enabled) continue
    if (provider.provider == 'openAI') {
      for (const model of (provider.models || [])) {
        const compositeId = createModelId(provider.id, model.api_model_name)
        options.push({
          label: `${model.api_model_name} (${provider.id})`,
          value: compositeId
        })
      }
    }

  }

  return options
})

async function loadSettings() {
  loading.value = true
  try {
    const response = await aiApi.getSettings()
    if (response.data) {
      // 迁移每个模型配置到新格式
      config.models = response.data.models || []
      config.global_config = {
        default_search_model_id: response.data.global_config?.default_search_model_id || '',
        default_tag_model_id: response.data.global_config?.default_tag_model_id || '',
        default_prompt_optimize_model_id: response.data.global_config?.default_prompt_optimize_model_id || '',
        prompt_optimize_system_prompt: response.data.global_config?.prompt_optimize_system_prompt || 'You are a prompt optimizer for image search. Convert Chinese queries into English descriptions for SigLIP model.\nRules:\n1. Translate Chinese to English accurately\n2. Expand into detailed visual descriptions (1-2 sentences)\n3. Include visual attributes: colors, lighting, style, mood\n4. Output ONLY the English prompt, nothing else\n\nExamples:\n- "日落海滩" -> "A beautiful sunset at the beach with warm orange and pink sky reflecting on calm ocean waves"\n- "可爱的猫咪" -> "An adorable cute cat with fluffy fur and expressive eyes in a cozy home setting"\n- "城市夜景" -> "Urban cityscape at night with illuminated skyscrapers and city lights"',
        default_naming_model_id: response.data.global_config?.default_naming_model_id || '',
        naming_system_prompt: response.data.global_config?.naming_system_prompt || 'You are an AI assistant specialized in naming photo albums. Based on the images in the album, generate a concise and descriptive Chinese name (2-8 characters).\nRules:\n1. Output ONLY the album name in Chinese, nothing else\n2. Keep it short (2-8 characters)\n3. Capture the main theme or emotion\n4. Be specific and descriptive\n\nExamples:\n- Images of sunset at beach -> "海边日落"\n- Images of cats -> "可爱猫咪"\n- Images of city nightscape -> "都市夜景"\n- Images of family gathering -> "家庭聚会"',
        naming_max_images: response.data.global_config?.naming_max_images || 3,
      }
    }
    // 保存原始数据
    Object.assign(originalConfig, JSON.parse(JSON.stringify(config)))
    console.log(embeddingModelOptions.value)
  } catch (error) {
    console.error('Failed to load AI settings:', error)
  } finally {
    loading.value = false
  }
}

function addProvider() {
  const newProvider: ModelConfig = {
    id: SELF_HOSTED_ID,
    models: [],
    endpoint: 'http://localhost:8100',
    api_key: '',
    provider: 'selfHosted',
    enabled: false,
  }
  config.models.push(newProvider)
}

function removeProvider(index: number) {
  config.models.splice(index, 1)
}

// 自托管模型的默认 ID
const SELF_HOSTED_ID = 'Self Hosted'

function onProviderChange(provider: ModelConfig) {
  if (provider.provider === 'selfHosted') {
    // 切换到自托管模型时，设置固定的 ID 和模型
    provider.id = SELF_HOSTED_ID
  } else {
    // 切换到非自托管模型时，清空模型列表
    if (provider.id === SELF_HOSTED_ID) {
      provider.id = `provider-${Date.now()}`
    }
    provider.models = []
  }
}

// 提供商选项
const providerOptions: SelectOption[] = [
  {label: '自托管模型', value: 'selfHosted'},
  {label: 'OpenAI 兼容', value: 'openAI'},
  {label: '阿里云Multimodal Embedding', value: 'alyunMultimodalEmbedding'}
]

// ================== 添加/编辑模型对话框 ==================

const showAddModelDialog = ref(false)
const currentProviderForAddModel = ref<ModelConfig | null>(null)
const editingModelIndex = ref<number | null>(null)  // null 表示添加模式，否则表示编辑模式
const newModel = reactive<ModelItem>({
  api_model_name: '',
  model_name: ''
})

// 是否处于编辑模式
const isEditMode = computed(() => editingModelIndex.value !== null)

function openAddModelDialog(provider: ModelConfig) {
  currentProviderForAddModel.value = provider
  editingModelIndex.value = null
  newModel.api_model_name = ''
  newModel.model_name = ''
  showAddModelDialog.value = true
}

function openEditModelDialog(provider: ModelConfig, index: number) {
  currentProviderForAddModel.value = provider
  editingModelIndex.value = index
  const model = provider.models?.[index]
  if (model) {
    newModel.api_model_name = model.api_model_name
    newModel.model_name = model.model_name
  }
  showAddModelDialog.value = true
}

function closeAddModelDialog() {
  showAddModelDialog.value = false
  currentProviderForAddModel.value = null
  editingModelIndex.value = null
}

function confirmAddModel() {
  if (!currentProviderForAddModel.value || !newModel.api_model_name) return

  if (!currentProviderForAddModel.value.models) {
    currentProviderForAddModel.value.models = []
  }

  if (isEditMode.value) {
    // 编辑模式：更新现有模型
    const idx = editingModelIndex.value!
    const existingModel = currentProviderForAddModel.value.models[idx]
    if (!existingModel) return

    // 检查是否与其他模型重名（排除自己）
    const exists = currentProviderForAddModel.value.models.some(
        (m, i) => i !== idx && m.api_model_name === newModel.api_model_name
    )
    if (exists) {
      dialogStore.notify({title: '提示', message: '该模型名称已存在', type: 'warning'})
      return
    }

    existingModel.api_model_name = newModel.api_model_name
    existingModel.model_name = newModel.model_name || newModel.api_model_name
  } else {
    // 添加模式：检查是否已存在
    const exists = currentProviderForAddModel.value.models.some(
        m => m.api_model_name === newModel.api_model_name
    )
    if (exists) {
      dialogStore.notify({title: '提示', message: '该模型已存在', type: 'warning'})
      return
    }

    currentProviderForAddModel.value.models.push({
      api_model_name: newModel.api_model_name,
      model_name: newModel.model_name || newModel.api_model_name
    })
  }

  closeAddModelDialog()
}

function removeModelFromProvider(provider: ModelConfig, index: number) {
  provider.models?.splice(index, 1)
}

// ================== 测试连接 ==================
const testingProvider = ref<string | null>(null)

async function testProviderConnection(provider: ModelConfig) {
  // 验证基本配置
  if (!provider.endpoint) {
    dialogStore.notify({title: '提示', message: '请先配置 API 端点', type: 'warning'})
    return
  }

  // 自托管模型不需要模型项，其他类型需要至少一个模型
  if (provider.provider !== 'selfHosted' && (!provider.models || provider.models.length === 0)) {
    dialogStore.notify({title: '提示', message: '请先添加至少一个模型', type: 'warning'})
    return
  }

  testingProvider.value = provider.id
  try {
    // 构建测试连接请求
    const request = {
      provider: provider,
      // 非自托管模型使用第一个模型进行测试
      model: provider.provider !== 'selfHosted' && provider.models?.length
          ? provider.models[0]
          : undefined
    }

    await aiApi.testConnection(request)
    dialogStore.notify({title: '成功', message: `${provider.id || '提供商'} 连接测试成功`, type: 'success'})
  } catch (error: any) {
    dialogStore.notify({title: '连接失败', message: error.message || '连接测试失败', type: 'error'})
  } finally {
    testingProvider.value = null
  }
}

async function handleSave() {
  // 验证：至少需要有一个模型
  if (config.models.length === 0) {
    dialogStore.notify({title: '提示', message: '请至少添加一个提供商配置', type: 'warning'})
    return
  }

  if (embeddingModelOptions.value.length > 0) {
    if (!config?.global_config?.default_search_model_id) {
      dialogStore.notify({title: '提示', message: '请设置默认的搜索模型', type: 'warning'})
      return
    }
    if (!config?.global_config?.default_tag_model_id) {
      dialogStore.notify({title: '提示', message: '请设置默认打标签模型', type: 'warning'})
      return
    }
  }
  if (llmsModelOptions.value.length > 0) {
    if (!config?.global_config?.default_prompt_optimize_model_id) {
      dialogStore.notify({title: '提示', message: '请设置默认的提示词优化模型', type: 'warning'})
      return
    }
  }


  saving.value = true
  emit('saving', true)
  try {
    await aiApi.updateSettings(config)
    // 更新原始数据
    Object.assign(originalConfig, JSON.parse(JSON.stringify(config)))
    emit('change', false)
    dialogStore.notify({title: '成功', message: 'AI 设置更新成功', type: 'success'})
  } catch (error: any) {
    dialogStore.notify({title: '错误', message: error.message || '更新 AI 设置失败', type: 'error'})
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
  Object.assign(config, JSON.parse(JSON.stringify(originalConfig)))
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
