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
                :options="enabledModelOptions"
                label="默认搜索模型"
                placeholder="选择默认搜索模型"
            />
            <p class="mt-1 text-xs text-gray-500">用于语义搜索的默认模型</p>
          </div>

          <!-- 默认打标签模型 -->
          <div>
            <BaseSelect
                v-model="config.global_config!.default_tag_model_id"
                :options="enabledModelOptions"
                label="默认打标签模型"
                placeholder="选择默认打标签模型"
            />
            <p class="mt-1 text-xs text-gray-500">用于自动生成图片标签的默认模型</p>
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
            @click="addModel"
            class="px-3 py-1.5 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 text-sm flex items-center gap-1"
        >
          <PlusIcon class="h-4 w-4"/>
          添加模型
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
            @click="addModel"
            class="px-4 py-2 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 text-sm"
        >
          添加第一个模型
        </button>
      </div>

      <div v-else class="divide-y divide-white/5">
        <div v-for="(model, index) in config.models" :key="model.id" class="p-6">
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-3">
              <!-- 启用开关 -->
              <button
                  @click="model.enabled = !model.enabled"
                  :class="[
                    'relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                    model.enabled ? 'bg-primary-500' : 'bg-gray-600'
                  ]"
              >
                <span
                    :class="[
                      'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                      model.enabled ? 'translate-x-4' : 'translate-x-0'
                    ]"
                />
              </button>
              <span class="text-white font-medium">{{ model.model_name || '未命名模型' }}</span>
              <span v-if="model.provider === 'selfHosted'"
                    class="px-2 py-0.5 rounded text-xs bg-amber-500/20 text-amber-400">自托管</span>
            </div>
            <div class="flex items-center gap-2">
              <button
                  @click="testModelConnection(model.id)"
                  :disabled="testing === model.id"
                  class="px-2 py-1 rounded text-xs text-gray-400 hover:text-green-400 hover:bg-white/5 transition-colors disabled:opacity-50"
              >
                {{ testing === model.id ? '测试中...' : '测试连接' }}
              </button>
              <button
                  @click="removeModel(index)"
                  class="p-1 rounded text-gray-400 hover:text-red-400 hover:bg-white/5 transition-colors"
              >
                <TrashIcon class="h-4 w-4"/>
              </button>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <BaseSelect
                  v-model="model.provider"
                  :options="providerOptions"
                  label="提供商"
                  @update:model-value="onProviderChange(model)"
              />
            </div>
            <div v-if="model.provider !== 'selfHosted'">
              <label class="block text-sm font-medium text-gray-400 mb-1.5">模型 ID</label>
              <input
                  v-model="model.id"
                  type="text"
                  placeholder="用于区分添加的不同模型"
                  class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-400 mb-1.5">API 端点</label>
              <input
                  v-model="model.endpoint"
                  type="text"
                  placeholder="如: http://localhost:8100"
                  class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm"
              />
            </div>
            <div v-if="model.provider !== 'selfHosted'">
              <label class="block text-sm font-medium text-gray-400 mb-1.5">api模型名称</label>
              <input
                  v-model="model.api_model_name"
                  @input="onApiModelNameChange(model)"
                  type="text"
                  placeholder="调用api时使用的模型名称，如: gemini"
                  class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm"
              />
            </div>
            <div v-if="model.provider !== 'selfHosted'">
              <label class="block text-sm font-medium text-gray-400 mb-1.5">模型名称(区分模型)</label>
              <input
                  v-model="model.model_name"
                  @input="userModifiedModelName[model.id] = model.model_name !== model.api_model_name;"
                  type="text"
                  placeholder="相通模型名称的provider将作为一组，被负载均衡器调用"
                  class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-400 mb-1.5">API Key</label>
              <input
                  v-model="model.api_key"
                  type="password"
                  placeholder="留空则无需认证"
                  class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm"
              />
            </div>

            <!-- 提示词优化配置（仅自托管模型） -->
            <div v-if="model.provider === 'selfHosted'" class="col-span-2 mt-1">
              <div class="flex items-center justify-between mb-3">
                <div>
                  <h4 class="text-sm font-medium text-white">提示词优化</h4>
                  <p class="text-xs text-gray-500">将中文搜索词转换为适合模型理解的英文描述</p>
                </div>
                <!-- 启用开关 -->
                <button
                    @click="togglePromptOptimizer(model)"
                    :class="[
                      'relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                      getPromptOptimizer(model).enabled ? 'bg-primary-500' : 'bg-gray-600'
                    ]"
                >
                  <span
                      :class="[
                        'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                        getPromptOptimizer(model).enabled ? 'translate-x-4' : 'translate-x-0'
                      ]"
                  />
                </button>
              </div>

              <!-- 系统提示词输入框 -->
              <div v-if="getPromptOptimizer(model).enabled">
                <label class="block text-sm font-medium text-gray-400 mb-1.5">系统提示词</label>
                <textarea
                    :value="getPromptOptimizer(model).system_prompt"
                    @input="updateSystemPrompt(model, ($event.target as HTMLTextAreaElement).value)"
                    placeholder="留空使用默认提示词..."
                    rows="4"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors text-sm resize-none"
                />
                <p class="mt-1 text-xs text-gray-500">默认提示词会将中文搜索词翻译为英文并扩展为详细的视觉描述</p>
              </div>
            </div>
          </div>
        </div>

        <!-- 保存按钮 -->
        <div class="p-4">
          <button
              @click="handleSave"
              :disabled="saving"
              class="px-6 py-2.5 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ saving ? '保存中...' : '保存 AI 设置' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import {onMounted, reactive, ref} from 'vue'
import {PlusIcon, TrashIcon} from '@heroicons/vue/24/outline'
import {useDialogStore} from '@/stores/dialog'
import type {AIConfig, ExtraConfig, ModelConfig, PromptOptimizerConfig} from '@/types/ai'
import type {SelectOption} from '@/components/common/BaseSelect.vue'
import BaseSelect from '@/components/common/BaseSelect.vue'
import {aiApi} from "@/api/ai.ts";

const dialogStore = useDialogStore()

const loading = ref(true)
const saving = ref(false)
const testing = ref<string | null>(null)

// 使用本地状态避免直接修改 store
const config = reactive<AIConfig>({
  models: [],
  global_config: {
    default_search_model_id: '',
    default_tag_model_id: ''
  }
})

// 计算启用的模型选项（用于全局设置下拉框）
const enabledModelOptions = ref<SelectOption[]>([])

async function loadSettings() {
  loading.value = true
  try {
    const response = await aiApi.getSettings()
    if (response.data) {
      config.models = response.data.models || []
      config.global_config = response.data.global_config || {
        default_search_model_id: config.models[0]?.id || '',
        default_tag_model_id: config.models[0]?.id || ''
      }
    }

    config.models.forEach(m => {
      enabledModelOptions.value.push({
        label: m.model_name || m.id,
        value: m.id
      })
    })
    console.log(enabledModelOptions.value)
  } catch (error) {
    console.error('Failed to load AI settings:', error)
  } finally {
    loading.value = false
  }
}

function addModel() {
  const newModel: ModelConfig = {
    id: `model-${Date.now()}`,
    api_model_name: '',
    model_name: '',
    endpoint: 'http://localhost:8100',
    api_key: '',
    provider: 'selfHosted',
    enabled: false,
  }
  config.models.push(newModel)
}

function removeModel(index: number) {
  const model = config.models[index]
  if (!model) return
  config.models.splice(index, 1)
}

// 自托管模型的默认 ID（模型名称由用户自行配置）
const SELF_HOSTED_ID = 'self-hosted'
const SELF_HOSTED_MODEL_NAME = 'google/siglip-so400m-patch14-384'

function onProviderChange(model: ModelConfig) {
  if (model.provider === 'selfHosted') {
    // 切换到自托管模型时，设置固定的 ID 和模型名称
    model.id = SELF_HOSTED_ID
    model.model_name = SELF_HOSTED_MODEL_NAME
    model.api_model_name = SELF_HOSTED_MODEL_NAME
  }
}

// 记录用户是否手动修改过 model_name
const userModifiedModelName = reactive<Record<string, boolean>>({})

// 提供商选项
const providerOptions: SelectOption[] = [
  {label: '自托管模型', value: 'selfHosted'},
  {label: 'OpenAI 兼容', value: 'openAI'},
  {label: '阿里云Multimodal Embedding', value: 'alyunMultimodalEmbedding'}
]

function onApiModelNameChange(model: ModelConfig) {
  // 如果用户没有手动修改过 model_name，则智能同步
  if (!userModifiedModelName[model.id]) {
    model.model_name = model.api_model_name
  }
}

// ================== 提示词优化器配置 ==================

// 解析 extra_config JSON
function parseExtraConfig(model: ModelConfig): ExtraConfig {
  if (!model.extra_config) return {}
  try {
    return JSON.parse(model.extra_config)
  } catch {
    return {}
  }
}

// 序列化 extra_config
function stringifyExtraConfig(config: ExtraConfig): string {
  return JSON.stringify(config)
}

// 获取提示词优化器配置
function getPromptOptimizer(model: ModelConfig): PromptOptimizerConfig {
  const extra = parseExtraConfig(model)
  return extra.prompt_optimizer ?? {enabled: true, system_prompt: ''}
}

// 更新提示词优化器配置
function updatePromptOptimizer(model: ModelConfig, config: Partial<PromptOptimizerConfig>) {
  const extra = parseExtraConfig(model)
  extra.prompt_optimizer = {
    ...getPromptOptimizer(model),
    ...config
  }
  model.extra_config = stringifyExtraConfig(extra)
}

// 切换提示词优化器启用状态
function togglePromptOptimizer(model: ModelConfig) {
  const current = getPromptOptimizer(model)
  updatePromptOptimizer(model, {enabled: !current.enabled})
}

// 更新系统提示词
function updateSystemPrompt(model: ModelConfig, value: string) {
  updatePromptOptimizer(model, {system_prompt: value})
}

async function testModelConnection(modelId: string) {
  testing.value = modelId
  try {
    const response = await aiApi.testConnection({id: modelId})
    dialogStore.notify({title: '成功', message: response.data?.message || '连接成功', type: 'success'})
  } catch (error: any) {
    dialogStore.notify({title: '连接失败', message: error.message || '无法连接到模型服务', type: 'error'})
  } finally {
    testing.value = null
  }
}

async function handleSave() {
  // 验证：至少需要有一个模型
  if (config.models.length === 0) {
    dialogStore.notify({title: '提示', message: '请至少添加一个模型配置', type: 'warning'})
    return
  }

  saving.value = true
  try {
    await aiApi.updateSettings(config)
    dialogStore.notify({title: '成功', message: 'AI 设置更新成功', type: 'success'})
  } catch (error: any) {
    dialogStore.notify({title: '错误', message: error.message || '更新 AI 设置失败', type: 'error'})
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>
