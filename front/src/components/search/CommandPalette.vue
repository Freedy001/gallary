<template>
  <Teleport to="body">
    <Transition name="command-palette">
      <div
          v-if="uiStore.commandPaletteOpen"
          class="fixed inset-0 z-50 flex items-start justify-center bg-black/60 backdrop-blur-sm p-4 pt-[10vh]"
          @click.self="close"
          @keydown.esc="close"
      >
        <div
            class="w-screen max-w-2xl overflow-hidden rounded-2xl border border-white/10 bg-[#0a0a0a]/90 shadow-[0_0_50px_-12px_rgba(0,0,0,0.8)] backdrop-blur-xl ring-1 ring-white/5"
            @click.stop>
          <!-- 搜索输入框 -->
          <div class="border-b border-white/5 px-5 py-5">
            <div class="flex items-center gap-4">
              <component
                  :is="isSemanticSearch ? SparklesIcon : MagnifyingGlassIcon"
                  :class="[
                    'h-6 w-6 shrink-0 animate-pulse',
                    isSemanticSearch ? 'text-pink-500' : 'text-primary-500'
                  ]"
              />
              <input
                  ref="searchInputRef"
                  v-model="imageStore.searchFilters.keyword"
                  type="text"
                  :placeholder="isSemanticSearch ? '描述你想找的图片，如：海边日落、穿红色衣服的人...' : '搜索影像记忆 / 日期 / 地点...'"
                  class="flex-1 border-none bg-transparent text-lg text-white placeholder:text-gray-600 focus:outline-none font-light tracking-wide"
                  @keydown.enter="executeSearch"
              />

              <!-- 提示词优化按钮 -->
              <tooltip :content="isOptimizing ? '正在优化...' : '优化提示词（将中文转换为更精确的英文描述）'">
                <button
                    v-if="isSemanticSearch && imageStore.searchFilters.keyword && hasChatCompletionModel"
                    :class="[
                    'flex items-center justify-center w-8 rounded-lg transition-all duration-200 shrink-0',
                    isOptimizing
                      ? 'bg-primary-500/20 cursor-wait'
                      : 'bg-white/5 hover:bg-primary-500/20 hover:text-primary-400'
                  ]"
                    :disabled="isOptimizing"
                    @click="optimizePrompt"
                >
                  <BoltIcon v-if="!isOptimizing" class="h-4 w-4"/>
                  <span v-else
                        class="h-4 w-4 animate-spin rounded-full border-2 border-primary-400 border-t-transparent"></span>
                </button>
              </tooltip>

              <!-- 嵌入模型选择器 -->
              <div v-if="isSemanticSearch && embeddingModels.length > 1" class="w-38">
                <BaseSelect
                    v-model="selectedEmbeddingModel"
                    :options="embeddingModelOptions"
                    placeholder="选择模型"
                    button-class="!py-1.5 !text-xs"
                />
              </div>

              <kbd
                  class="rounded-md bg-white/10 px-2 py-1 text-xs font-mono text-gray-400 border border-white/5">ESC</kbd>
            </div>
          </div>

          <!-- 筛选选项 -->
          <div class="border-b border-white/5 px-5 py-4 bg-white/2">
            <div class="flex flex-wrap gap-2">
              <!-- AI 语义搜索 -->
              <button
                  v-if="hasEmbeddingModel"
                  @click="isSemanticSearch = !isSemanticSearch"
                  :class="[
                  'flex items-center gap-1.5 rounded-full px-4 py-1.5 text-xs font-medium transition-all duration-300 border',
                  isSemanticSearch
                    ? 'bg-linear-to-r from-primary-500/30 to-pink-500/30 text-primary-200 border-primary-400/50 shadow-glow'
                    : 'bg-white/5 text-gray-400 border-transparent hover:bg-white/10 hover:text-gray-200',
                ]"
              >
                <SparklesIcon class="h-3.5 w-3.5"/>
                AI 语义搜索
              </button>


              <!-- 日期范围 -->
              <button
                  @click="toggleFilter('date')"
                  :class="[
                  'flex items-center gap-1.5 rounded-full px-4 py-1.5 text-xs font-medium transition-all duration-300 border',
                  activeFilters.has('date')
                    ? 'bg-primary-500/20 text-primary-300 border-primary-500/30 shadow-[0_0_10px_rgba(139,92,246,0.2)]'
                    : 'bg-white/5 text-gray-400 border-transparent hover:bg-white/10 hover:text-gray-200',
                ]"
              >
                <CalendarIcon class="h-3.5 w-3.5"/>
                日期范围
              </button>


              <!-- GPS位置 -->
              <button
                  @click="toggleFilter('location')"
                  :class="[
                  'flex items-center gap-1.5 rounded-full px-4 py-1.5 text-xs font-medium transition-all duration-300 border',
                  activeFilters.has('location')
                    ? 'bg-primary-500/20 text-primary-300 border-primary-500/30 shadow-[0_0_10px_rgba(139,92,246,0.2)]'
                    : 'bg-white/5 text-gray-400 border-transparent hover:bg-white/10 hover:text-gray-200',
                ]"
              >
                <MapPinIcon class="h-3.5 w-3.5"/>
                地理位置
              </button>

              <!-- 标签 -->
              <button
                  @click="toggleFilter('tags')"
                  :class="[
                  'flex items-center gap-1.5 rounded-full px-4 py-1.5 text-xs font-medium transition-all duration-300 border',
                  activeFilters.has('tags')
                    ? 'bg-primary-500/20 text-primary-300 border-primary-500/30 shadow-[0_0_10px_rgba(139,92,246,0.2)]'
                    : 'bg-white/5 text-gray-400 border-transparent hover:bg-white/10 hover:text-gray-200',
                ]"
              >
                <TagIcon class="h-3.5 w-3.5"/>
                智能标签
              </button>
            </div>
          </div>

          <!-- 图片上传区域（仅在语义搜索开启时显示） -->
          <div v-if="isSemanticSearch" class="border-b border-white/5 px-5 py-4 bg-black/20 ">
            <label class="mb-3 block text-xs font-medium text-gray-400">以图搜图（可选）</label>

            <!-- 已选择图片预览 -->
            <div v-if="searchImagePreview" class="relative inline-block">
              <img
                  :src="searchImagePreview"
                  alt="搜索图片"
                  class="h-24 w-24 object-cover rounded-xl border border-white/10"
              />
              <button
                  @click="removeSearchImage"
                  class="absolute -top-2 -right-2 p-1 rounded-full bg-red-500/80 text-white hover:bg-red-500 transition-colors"
              >
                <XMarkIcon class="h-4 w-4"/>
              </button>
            </div>

            <!-- 图片上传区域 -->
            <div
                v-else
                @drop="handleDrop"
                @dragover="handleDragOver"
                @dragleave="handleDragLeave"
                @click="imageInputRef?.click()"
                :class="[
                  'flex items-center justify-center gap-3 h-24 rounded-xl border-2 border-dashed cursor-pointer transition-all duration-200',
                  isDragging
                    ? 'border-primary-500 bg-primary-500/10'
                    : 'border-white/10 hover:border-white/20 hover:bg-white/2'
                ]"
            >
              <PhotoIcon class="h-8 w-8 text-gray-500"/>
              <div class="text-center">
                <p class="text-sm text-gray-400">拖拽、粘贴图片或点击上传</p>
                <p class="text-xs text-gray-600 mt-1">支持 Ctrl+V 粘贴剪贴板图片</p>
              </div>
            </div>

            <input
                ref="imageInputRef"
                type="file"
                accept="image/*"
                class="hidden"
                @change="handleImageSelect"
            />
          </div>

          <!-- 筛选器详细配置 -->
          <div v-if="activeFilters.size > 0"
               class="border-b border-white/5 bg-black/20 px-5 py-6 animate-slide-in-top max-h-[50vh] overflow-y-auto custom-scrollbar">
            <!-- 日期筛选 -->
            <div v-if="activeFilters.has('date')" class="mb-6 last:mb-0">
              <label class="mb-3 block text-sm font-medium text-gray-300">日期范围</label>
              <div class="flex items-center gap-3">
                <input
                    v-model="imageStore.searchFilters.start_date"
                    type="date"
                    class="flex-1 rounded-xl border border-white/10 bg-white/5 px-4 py-2.5 text-sm text-white transition-colors focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
                />
                <span class="text-sm font-medium text-gray-600">至</span>
                <input
                    v-model="imageStore.searchFilters.end_date"
                    type="date"
                    class="flex-1 rounded-xl border border-white/10 bg-white/5 px-4 py-2.5 text-sm text-white transition-colors focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
                />
              </div>
            </div>

            <!-- 位置筛选 -->
            <div v-if="activeFilters.has('location')" class="mb-6 last:mb-0">
              <LocationPicker
                  v-model="imageStore.searchFilters.location"
                  v-model:latitude="imageStore.searchFilters.latitude"
                  v-model:longitude="imageStore.searchFilters.longitude"
                  label="位置名称 (搜索)"
                  :show-map="true"
                  placeholder="例如: 北京"
              />
              <!-- 搜索半径选择 -->
              <div class="mt-4">
                <label class="block text-sm font-medium text-white/80 mb-2">
                  搜索半径: <span class="text-primary-400">{{ imageStore.searchFilters.radius || 10 }} 公里</span>
                </label>
                <div class="flex items-center gap-3">
                  <span class="text-xs text-gray-500">1km</span>
                  <input
                      type="range"
                      min="1"
                      max="100"
                      :value="imageStore.searchFilters.radius || 10"
                      @input="(e) => imageStore.searchFilters.radius = Number((e.target as HTMLInputElement).value)"
                      class="flex-1 cursor-pointer accent-primary-500 h-1.5 bg-white/10 rounded-full appearance-none hover:bg-white/20"
                  />
                  <span class="text-xs text-gray-500">100km</span>
                </div>
              </div>
            </div>

            <!-- 标签筛选 -->
            <div v-if="activeFilters.has('tags')" class="mb-6 last:mb-0">
              <label class="mb-3 block text-sm font-medium text-gray-300">标签</label>

              <!-- 已选中的标签 -->
              <div v-if="selectedTags.length > 0" class="flex flex-wrap gap-2 mb-3">
                <span
                    v-for="tag in selectedTags"
                    :key="tag.id"
                    class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-primary-500/20 text-primary-300 border border-primary-500/30"
                >
                  {{ tag.name }}
                  <button
                      @click="removeTag(tag.id)"
                      class="hover:text-white transition-colors"
                  >
                    <XMarkIcon class="h-3.5 w-3.5"/>
                  </button>
                </span>
              </div>

              <!-- 搜索输入框 -->
              <div class="relative">
                <input
                    v-model="tagSearchQuery"
                    @focus="tagDropdownOpen = true"
                    type="text"
                    placeholder="搜索标签..."
                    class="w-full rounded-xl border border-white/10 bg-white/5 px-4 py-2.5 text-sm text-white transition-colors placeholder:text-gray-600 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
                />

                <!-- 标签下拉列表 -->
                <div
                    v-if="tagDropdownOpen && filteredTags.length > 0"
                    class="relative z-10 mt-2 w-full max-h-48 overflow-y-auto rounded-xl border border-white/10 bg-[#1a1a1a] shadow-lg"
                >
                  <button
                      v-for="tag in filteredTags"
                      :key="tag.id"
                      @click="toggleTag(tag.id)"
                      class="flex items-center justify-between w-full px-4 py-2.5 text-sm text-left hover:bg-white/5 transition-colors"
                      :class="imageStore.searchFilters.tags?.includes(tag.id) ? 'text-primary-300' : 'text-gray-300'"
                  >
                    <span>{{ tag.name }}</span>
                    <CheckIcon v-if="imageStore.searchFilters.tags?.includes(tag.id)" class="h-4 w-4 text-primary-400"/>
                  </button>
                </div>
              </div>

              <!-- 点击外部关闭下拉 -->
              <div
                  v-if="tagDropdownOpen"
                  class="fixed inset-0 z-0"
                  @click="tagDropdownOpen = false"
              />
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex items-center justify-between bg-white/5 px-5 py-4 backdrop-blur-md">
            <button
                @click="clearFilters"
                class="text-sm font-medium text-gray-500 transition-colors hover:text-gray-300"
            >
              清除所有筛选
            </button>

            <div class="flex gap-3">
              <button
                  @click="close"
                  class="rounded-xl border border-white/10 bg-transparent px-5 py-2 text-sm font-medium text-gray-400 transition-all hover:bg-white/5 hover:text-white"
              >
                取消
              </button>
              <button
                  @click="executeSearch"
                  :disabled="semanticSearching"
                  :class="[
                    'rounded-xl px-6 py-2 text-sm font-bold text-white transition-all active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed',
                    isSemanticSearch
                      ? 'bg-linear-to-r from-primary-600 to-pink-600 shadow-[0_0_20px_rgba(236,72,153,0.4)] hover:shadow-[0_0_30px_rgba(236,72,153,0.6)]'
                      : 'bg-primary-600 shadow-[0_0_20px_rgba(124,58,237,0.4)] hover:bg-primary-500 hover:shadow-[0_0_30px_rgba(124,58,237,0.6)]'
                  ]"
              >
                <span v-if="semanticSearching" class="flex items-center gap-2">
                  <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                  搜索中...
                </span>
                <span v-else-if="isSemanticSearch" class="flex items-center gap-1.5">
                  <SparklesIcon class="h-4 w-4"/>
                  语义搜索
                </span>
                <span v-else>搜索影像</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import {computed, nextTick, onMounted, onUnmounted, ref, watch} from 'vue'
import {useUIStore} from '@/stores/ui'
import LocationPicker from '@/components/common/LocationPicker.vue'
import type {SelectOption} from '@/components/common/BaseSelect.vue'
import BaseSelect from '@/components/common/BaseSelect.vue'
import {
  BoltIcon,
  CalendarIcon,
  CheckIcon,
  MagnifyingGlassIcon,
  MapPinIcon,
  PhotoIcon,
  SparklesIcon,
  TagIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import type {SearchParams, Tag} from '@/types'
import type {EmbeddingModelInfo} from '@/types/ai'
import {imageApi} from "@/api/image.ts"
import {aiApi} from "@/api/ai.ts"
import {useDialogStore} from "@/stores/dialog.ts";
import Tooltip from "@/components/common/Tooltip.vue";
import {useSearchStore} from "@/stores/search.ts";

const uiStore = useUIStore()
const dialogStore = useDialogStore();

const searchInputRef = ref<HTMLInputElement>()
const isSemanticSearch = ref(false)
const semanticSearching = ref(false)
const imageStore = useSearchStore();

// 图片搜索相关状态
const searchImage = ref<File | null>(null)
const searchImagePreview = ref<string>('')
const isDragging = ref(false)
const imageInputRef = ref<HTMLInputElement>()

const activeFilters = ref(new Set<string>())

// 标签相关状态
const allTags = ref<Tag[]>([])
const selectedTagsMap = ref<Map<number, Tag>>(new Map()) // 保存已选中标签的完整信息
const tagSearchQuery = ref('')
const tagDropdownOpen = ref(false)
const isLoadingTags = ref(false)

// 过滤后的标签列表（排除已选中的）
const filteredTags = computed(() => {
  return allTags.value.filter(tag => !selectedTagsMap.value.has(tag.id))
})

// 获取已选中的标签对象
const selectedTags = computed(() => {
  return Array.from(selectedTagsMap.value.values())
})

// 防抖搜索标签
let tagSearchTimer: ReturnType<typeof setTimeout> | null = null

// 监听标签搜索输入
watch(tagSearchQuery, (keyword: string) => {
  if (tagSearchTimer) clearTimeout(tagSearchTimer)
  tagSearchTimer = setTimeout(async () => {
    await loadTags(keyword)
  }, 300)
})

// 加载标签列表
async function loadTags(keyword?: string) {
  isLoadingTags.value = true
  try {
    const response = await imageApi.getTags(keyword, 20)
    if (response.data) {
      allTags.value = response.data
    }
  } catch (error) {
    console.error('加载标签列表失败:', error)
  } finally {
    isLoadingTags.value = false
  }
}

// 切换标签选中状态
function toggleTag(tagId: number) {
  const tags = imageStore.searchFilters.tags || []
  const index = tags.indexOf(tagId)
  if (index === -1) {
    // 添加标签
    imageStore.searchFilters.tags = [...tags, tagId]
    // 保存标签完整信息到 map
    const tag = allTags.value.find(t => t.id === tagId)
    if (tag) {
      selectedTagsMap.value.set(tagId, tag)
    }
  } else {
    // 移除标签
    imageStore.searchFilters.tags = tags.filter(id => id !== tagId)
    selectedTagsMap.value.delete(tagId)
  }
}

// 移除标签
function removeTag(tagId: number) {
  imageStore.searchFilters.tags = (imageStore.searchFilters.tags || []).filter(id => id !== tagId)
  selectedTagsMap.value.delete(tagId)
}

// 嵌入模型相关状态
const embeddingModels = ref<EmbeddingModelInfo[]>([])
const selectedEmbeddingModel = ref<string>('')  // 存储 model_name，用于 API 调用


const embeddingModelOptions = computed<SelectOption[]>(() => {
  return embeddingModels.value.map(model => ({
    label: `${model.model_name}(${model.provider_id})`,  // 显示完整的模型 ID
    value: `${model.provider_id},${model.model_name}`    // 值使用 model_name（用于 API 调用）
  }))
})

// 是否有可用的嵌入模型
const hasEmbeddingModel = computed(() => {
  return embeddingModels.value.length > 0
})

// ChatCompletion 模型相关状态
const isOptimizing = ref(false)

// 是否有可用的 ChatCompletion 模型
const hasChatCompletionModel = ref(false)

let first = true

// 加载 ChatCompletion 模型列表
async function loadChatCompletionModels() {
  try {
    const response = await aiApi.configedDefaultModel("DefaultPromptOptimizeModelId")
    hasChatCompletionModel.value = response.data
  } catch (error) {
    dialogStore.notify({
      title: '加载 ChatCompletion 模型列表失败:',
      message: (error as Error).message,
      type: 'error'
    })
  }
}

// 优化提示词
async function optimizePrompt() {
  const query = imageStore.searchFilters.keyword?.trim()
  if (!query || isOptimizing.value) return

  isOptimizing.value = true
  try {
    const response = await aiApi.optimizePrompt({query})
    if (response.data?.optimized_prompt) {
      // 将优化后的提示词填充到搜索框
      imageStore.searchFilters.keyword = response.data.optimized_prompt

      dialogStore.notify({
        title: '提示词已优化',
        message: `"${query}" → "${response.data.optimized_prompt}"`,
        type: 'success'
      })
    }
  } catch (error) {
    dialogStore.notify({
      title: '优化失败',
      message: (error as Error).message,
      type: 'error'
    })
  } finally {
    isOptimizing.value = false
  }
}

// 加载嵌入模型列表
async function loadEmbeddingModels() {
  try {
    const response = await aiApi.getEmbeddingModels()
    if (response.data) {
      embeddingModels.value = response.data
      // 自动选择第一个模型
      if (response.data.length > 0) {
        const modelId = embeddingModelOptions.value[0]?.value;
        if (typeof modelId === 'string') selectedEmbeddingModel.value = modelId
        // 如果存在嵌入模型，默认开启语义搜索
        if (first && !isSemanticSearch.value) {
          isSemanticSearch.value = true
          first = false
        }
      }
    }
  } catch (error) {
    console.error('加载嵌入模型列表失败:', error)
  }
}

// 图片搜索相关函数
function handleImageSelect(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.files && input.files[0]) {
    setSearchImage(input.files[0])
  }
}

function handleDrop(event: DragEvent) {
  event.preventDefault()
  isDragging.value = false
  if (event.dataTransfer?.files && event.dataTransfer.files[0]) {
    const file = event.dataTransfer.files[0]
    if (file.type.startsWith('image/')) {
      setSearchImage(file)
    }
  }
}

function handleDragOver(event: DragEvent) {
  event.preventDefault()
  isDragging.value = true
}

function handleDragLeave() {
  isDragging.value = false
}

function handlePaste(event: ClipboardEvent) {
  // 仅在语义搜索模式下处理粘贴
  if (!isSemanticSearch.value || !uiStore.commandPaletteOpen) return

  const items = event.clipboardData?.items
  if (!items) return

  for (const item of items) {
    if (item.type.startsWith('image/')) {
      event.preventDefault()
      const file = item.getAsFile()
      if (file) {
        setSearchImage(file)
      }
      break
    }
  }
}

function setSearchImage(file: File) {
  searchImage.value = file
  // 创建预览 URL
  if (searchImagePreview.value) {
    URL.revokeObjectURL(searchImagePreview.value)
  }
  searchImagePreview.value = URL.createObjectURL(file)
}

function removeSearchImage() {
  searchImage.value = null
  if (searchImagePreview.value) {
    URL.revokeObjectURL(searchImagePreview.value)
    searchImagePreview.value = ''
  }
  if (imageInputRef.value) {
    imageInputRef.value.value = ''
  }
}

// 监听命令面板打开，自动聚焦输入框
watch(() => uiStore.commandPaletteOpen, (isOpen) => {
  if (isOpen) {
    nextTick(() => {
      searchInputRef.value?.focus()
    })
    // 加载嵌入模型列表
    loadEmbeddingModels()
    // 加载 ChatCompletion 模型列表
    loadChatCompletionModels()
    // 加载标签列表
    loadTags()
  }
})

function toggleFilter(filterName: string) {
  if (activeFilters.value.has(filterName)) {
    activeFilters.value.delete(filterName)
  } else {
    activeFilters.value.add(filterName)
  }
}

function clearFilters() {
  activeFilters.value.clear()
  tagSearchQuery.value = ''
  tagDropdownOpen.value = false
  selectedTagsMap.value.clear()
  removeSearchImage()
  imageStore.searchFilters = {
    keyword: imageStore.searchFilters.keyword,
    start_date: '',
    end_date: '',
    location: '',
    tags: [],
    latitude: undefined,
    longitude: undefined,
    radius: 10,
  }
}

async function executeSearch() {
  // 构建搜索参数
  const searchParams: SearchParams = {}

  // 始终添加传统筛选条件（如果有）
  if (imageStore.searchFilters.keyword) searchParams.keyword = imageStore.searchFilters.keyword
  if (imageStore.searchFilters.start_date) searchParams.start_date = imageStore.searchFilters.start_date
  if (imageStore.searchFilters.end_date) searchParams.end_date = imageStore.searchFilters.end_date
  if (imageStore.searchFilters.location) searchParams.location = imageStore.searchFilters.location
  if (imageStore.searchFilters.tags && imageStore.searchFilters.tags.length > 0) searchParams.tags = imageStore.searchFilters.tags
  // 经纬度搜索（优先使用经纬度，如果有的话）
  if (imageStore.searchFilters.latitude !== undefined && imageStore.searchFilters.longitude !== undefined) {
    searchParams.latitude = imageStore.searchFilters.latitude
    searchParams.longitude = imageStore.searchFilters.longitude
    searchParams.radius = imageStore.searchFilters.radius || 10
  }

  // 如果启用语义搜索，添加语义搜索参数（与传统筛选条件组合使用）
  if (isSemanticSearch.value) {
    searchParams.model_id = selectedEmbeddingModel.value
    searchParams.page_size = 50
  }

  // 如果有搜索图片，添加到参数中
  if (searchImage.value) {
    searchParams.file = searchImage.value
  }

  // 构建搜索描述
  const parts = []
  if (searchImage.value) {
    parts.push('📷 以图搜图')
  }
  if (imageStore.searchFilters.keyword) {
    parts.push(isSemanticSearch.value ? `AI: "${imageStore.searchFilters.keyword.trim()}"` : `关键词: "${imageStore.searchFilters.keyword}"`)
  }
  if (imageStore.searchFilters.start_date || imageStore.searchFilters.end_date) {
    parts.push(`日期: ${imageStore.searchFilters.start_date || '开始'} - ${imageStore.searchFilters.end_date || '至今'}`)
  }
  if (imageStore.searchFilters.location) parts.push(`位置: "${imageStore.searchFilters.location}"`)
  if (imageStore.searchFilters.tags && imageStore.searchFilters.tags.length > 0) {
    const tagNames = selectedTags.value.map(t => t.name).join(', ')
    parts.push(`标签: "${tagNames}"`)
  }

  // 通过 emit 通知父组件执行搜索
  imageStore.callSubscribers(searchParams, parts.join(' | ') || '搜索结果')
  close()
}


function close() {
  uiStore.closeCommandPalette()
}

// 键盘快捷键
function handleKeydown(event: KeyboardEvent) {
  // Cmd/Ctrl + K 打开命令面板
  if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
    event.preventDefault()
    uiStore.toggleCommandPalette()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  window.addEventListener('paste', handlePaste)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('paste', handlePaste)
})
</script>
