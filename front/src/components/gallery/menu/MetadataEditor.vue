<template>
  <Modal
      :model-value="modelValue"
      :title="title"
      size="lg"
      @update:model-value="close"
  >
    <form @submit.prevent="save" class="space-y-4">
      <!-- Single Image Fields -->
      <div v-if="isSingleMode">
        <label class="block text-sm font-medium text-white/80 mb-2">文件名</label>
        <input
            v-model="form.original_name"
            type="text"
            class="glass-input"
            placeholder="输入文件名称"
        />
      </div>

      <!-- 拍摄时间 -->
      <div>
        <label class="block text-sm font-medium text-white/80 mb-2">拍摄时间</label>
        <div class="relative">
          <input
              v-model="form.taken_at"
              type="datetime-local"
              class="glass-input datetime-input w-full"
              placeholder="选择拍摄时间"
          />
          <div class="absolute inset-y-0 right-0 flex items-center pr-3 pointer-events-none text-white/40">
            <!-- 这里可以放一个日历图标，如果原生图标不可见的话 -->
          </div>
        </div>
      </div>

      <!-- Location and Map -->
      <div class="space-y-4">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <!-- 使用新的 LocationPicker 组件 -->
          <div class="col-span-2 sm:col-span-2">
             <LocationPicker
               v-model="form.location_name"
               v-model:latitude="form.latitude"
               v-model:longitude="form.longitude"
               label="地点名称"
               :show-map="true"
             />
          </div>

          <!-- 显示经纬度 (只读) -->
          <div class="col-span-2 sm:col-span-2">
            <label class="block text-sm font-medium text-white/80 mb-2">经纬度 (自动获取)</label>
            <div class="flex space-x-2">
              <input
                  v-model.number="form.latitude"
                  type="number"
                  step="any"
                  class="glass-input bg-white/5 cursor-not-allowed"
                  placeholder="纬度"
                  readonly
              />
              <input
                  v-model.number="form.longitude"
                  type="number"
                  step="any"
                  class="glass-input bg-white/5 cursor-not-allowed"
                  placeholder="经度"
                  readonly
              />
            </div>
          </div>
        </div>
      </div>

      <!-- 标签选择器 -->
      <div>
        <label class="block text-sm font-medium text-white/80 mb-2">标签</label>

        <!-- 已选标签展示 -->
        <div v-if="selectedTags.length > 0" class="flex flex-wrap gap-2 mb-2">
          <span
              v-for="tag in selectedTags"
              :key="tag.id || tag.name"
              class="tag-badge"
              :style="tag.color ? { backgroundColor: tag.color + '33', borderColor: tag.color } : {}"
          >
            {{ tag.name }}
            <button
                type="button"
                @click="removeTag(tag)"
                class="ml-1 hover:text-red-400 transition-colors"
            >
              <XMarkIcon class="h-3 w-3"/>
            </button>
          </span>
        </div>

        <!-- 标签输入和下拉选择 -->
        <div class="relative">
          <input
              v-model="tagSearchQuery"
              type="text"
              class="glass-input"
              placeholder="搜索或创建标签..."
              @focus="showTagDropdown = true"
              @input="onTagInput"
              @keydown.enter.prevent="handleTagEnter"
              @keydown.escape="showTagDropdown = false"
          />

          <!-- 下拉菜单 -->
          <div
              v-if="showTagDropdown && (filteredTags.length > 0 || tagSearchQuery.trim())"
              class="tag-dropdown"
          >
            <!-- 现有标签列表 -->
            <div
                v-for="tag in filteredTags"
                :key="tag.id"
                class="tag-option"
                @click="selectTag(tag)"
            >
              <span
                  class="tag-color-dot"
                  :style="{ backgroundColor: tag.color || '#6366f1' }"
              ></span>
              {{ tag.name }}
            </div>


            <!-- 无结果提示 -->
            <div v-if="filteredTags.length === 0" class="tag-no-result">
              {{ tagSearchQuery.trim() ? '未找到匹配的标签' : '暂无标签' }}
            </div>
          </div>
        </div>

        <!-- 点击外部关闭下拉 -->
        <div
            v-if="showTagDropdown"
            class="fixed inset-0 z-40"
            @click="showTagDropdown = false"
        ></div>
      </div>

      <!-- Metadata Key-Value Pairs -->
      <div>
        <div class="flex justify-between items-center mb-2">
          <label class="block text-sm font-medium text-white/80">扩展元数据</label>
          <button
              type="button"
              @click="addMetadataField"
              class="text-sm text-blue-400 hover:text-blue-300 transition-colors"
          >
            + 添加字段
          </button>
        </div>
        <div v-for="(item, index) in form.metadata" :key="index" class="flex space-x-2 mb-2">
          <input
              v-model="item.key"
              type="text"
              class="glass-input"
              placeholder="键名"
          />
          <input
              v-model="item.value"
              type="text"
              class="glass-input"
              placeholder="键值"
          />
          <button
              type="button"
              @click="removeMetadataField(index)"
              class="text-red-400 hover:text-red-300 transition-colors p-2"
          >
            <XMarkIcon class="h-5 w-5"/>
          </button>
        </div>
      </div>

    </form>
    <template #footer>
      <div class="flex justify-end space-x-3">
        <button
            type="button"
            class="glass-button-secondary"
            @click="close"
        >
          取消
        </button>
        <button
            type="button"
            class="glass-button-primary"
            @click="save"
            :disabled="loading"
        >
          {{ loading ? '保存中...' : '保存' }}
        </button>
      </div>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import Modal from '@/components/common/Modal.vue'
import LocationPicker from '@/components/common/LocationPicker.vue'
import {XMarkIcon} from '@heroicons/vue/24/outline'
import type {Image, MetadataUpdate, Tag, UpdateMetadataRequest} from '@/types'
import {imageApi} from '@/api/image.ts'

const props = defineProps<{
  modelValue: boolean
  imageIds: number[]
  initialData?: Image | null // For single edit pre-fill
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved'): void
}>()

const loading = ref(false)

// 标签相关状态
const allTags = ref<Tag[]>([])
const selectedTags = ref<Tag[]>([])
const tagSearchQuery = ref('')
const showTagDropdown = ref(false)
const isLoadingTags = ref(false)

interface FormState {
  original_name?: string
  taken_at?: string
  location_name?: string
  latitude?: number
  longitude?: number
  metadata: MetadataUpdate[]
}

const form = ref<FormState>({
  metadata: []
})

const isSingleMode = computed(() => props.imageIds.length === 1)
const title = computed(() => isSingleMode.value ? '编辑图片元数据' : `批量编辑 ${props.imageIds.length} 张图片`)

// 标签过滤（排除已选中的）
const filteredTags = computed(() => {
  const selectedIds = new Set(selectedTags.value.map(t => t.id))
  return allTags.value.filter(tag => !selectedIds.has(tag.id))
})

// 防抖搜索标签
let tagSearchTimer: ReturnType<typeof setTimeout> | null = null
// 监听标签搜索输入
watch(tagSearchQuery, (keyword) => {
  if (tagSearchTimer) clearTimeout(tagSearchTimer)
  tagSearchTimer = setTimeout(async () => {
    await loadTags(keyword)
  }, 300)
})

// 加载标签列表
const loadTags = async (keyword?: string) => {
  isLoadingTags.value = true
  try {
    const res = await imageApi.getTags(keyword, 20)
    if (res.data) {
      allTags.value = res.data
    }
  } catch (e) {
    console.error('Failed to load tags', e)
  } finally {
    isLoadingTags.value = false
  }
}

// 选择已有标签
const selectTag = (tag: Tag) => {
  if (!selectedTags.value.some(t => t.id === tag.id)) {
    selectedTags.value.push(tag)
  }
  tagSearchQuery.value = ''
  showTagDropdown.value = false
}

// 移除标签
const removeTag = (tag: Tag) => {
  selectedTags.value = selectedTags.value.filter(t => t.id !== tag.id)
}

// 处理输入
const onTagInput = () => {
  showTagDropdown.value = true
}

// 处理回车键
const handleTagEnter = () => {
  const query = tagSearchQuery.value.trim()
  if (!query) return

  // 如果有匹配的标签，选择第一个
  const firstMatch = filteredTags.value[0]
  if (firstMatch) {
    selectTag(firstMatch)
  }
}

// Reset form when opening
watch(() => props.modelValue, async (val) => {
  if (val) {
    // 加载标签列表
    await loadTags()

    if (isSingleMode.value && props.initialData) {
      // Pre-fill for single image
      form.value = {
        original_name: props.initialData.original_name,
        taken_at: props.initialData.taken_at ? formatDatetimeLocal(props.initialData.taken_at) : undefined,
        location_name: props.initialData.location_name || undefined,
        latitude: props.initialData.latitude || undefined,
        longitude: props.initialData.longitude || undefined,
        metadata: props.initialData.metadata?.map(m => ({
          key: m.meta_key,
          value: m.meta_value,
          value_type: m.value_type
        })) || []
      }
      // 设置已选标签
      selectedTags.value = props.initialData.tags ? [...props.initialData.tags] : []
    } else {
      // Empty for batch edit
      form.value = {
        metadata: []
      }
      selectedTags.value = []
    }
    tagSearchQuery.value = ''
    showTagDropdown.value = false
  }
})

// 将 ISO 8601 日期时间格式转换为 datetime-local 输入所需的格式
const formatDatetimeLocal = (isoString: string): string => {
  const date = new Date(isoString)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hours}:${minutes}`
}

const addMetadataField = () => {
  form.value.metadata.push({key: '', value: '', value_type: 'string'})
}

const removeMetadataField = (index: number) => {
  form.value.metadata.splice(index, 1)
}

const close = () => {
  emit('update:modelValue', false)
}

const save = async () => {
  try {
    loading.value = true

    const data: UpdateMetadataRequest = {
      image_ids: props.imageIds,
      ...form.value,
      // 将选中的标签转换为名称数组
      tags: selectedTags.value.map(t => t.name),
    }

    // 将 datetime-local 格式转换为 ISO 8601 格式
    if (data.taken_at) {
      data.taken_at = new Date(data.taken_at).toISOString()
    }

    // Filter out empty metadata
    data.metadata = data.metadata?.filter(m => m.key)

    await imageApi.updateMetadata(data)
    emit('saved')
    close()
  } catch (e) {
    console.error('Failed to update metadata', e)
    // Handle error (could add a toast here if available)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.glass-input {
  width: 100%;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 12px;
  padding: 12px 16px;
  color: white;
  font-size: 14px;
  outline: none;
  transition: all 0.2s ease;
}

.glass-input::placeholder {
  color: rgba(255, 255, 255, 0.4);
}

.glass-input:focus {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.25);
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.05);
}

/* datetime-local 输入框特殊样式 */
.datetime-input {
  color-scheme: dark;
  cursor: pointer;
}

.datetime-input::-webkit-calendar-picker-indicator {
  filter: invert(1);
  opacity: 0.6;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s ease;
  position: absolute;
  right: 12px;
  height: 20px;
  width: 20px;
}

.datetime-input::-webkit-calendar-picker-indicator:hover {
  opacity: 1;
  background: rgba(255, 255, 255, 0.1);
}

.datetime-input::-webkit-datetime-edit {
  padding: 0;
}

.datetime-input::-webkit-datetime-edit-fields-wrapper {
  padding: 0;
}

.datetime-input::-webkit-datetime-edit-text {
  color: rgba(255, 255, 255, 0.6);
  padding: 0 2px;
}

.datetime-input::-webkit-datetime-edit-month-field,
.datetime-input::-webkit-datetime-edit-day-field,
.datetime-input::-webkit-datetime-edit-year-field,
.datetime-input::-webkit-datetime-edit-hour-field,
.datetime-input::-webkit-datetime-edit-minute-field {
  color: white;
  padding: 2px;
  border-radius: 4px;
}

.datetime-input::-webkit-datetime-edit-month-field:focus,
.datetime-input::-webkit-datetime-edit-day-field:focus,
.datetime-input::-webkit-datetime-edit-year-field:focus,
.datetime-input::-webkit-datetime-edit-hour-field:focus,
.datetime-input::-webkit-datetime-edit-minute-field:focus {
  background: rgba(59, 130, 246, 0.3);
  color: white;
  outline: none;
}

.glass-button-primary {
  padding: 10px 20px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 500;
  color: white;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.8), rgba(99, 102, 241, 0.8));
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.2s ease;
  backdrop-filter: blur(8px);
}

.glass-button-primary:hover:not(:disabled) {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.9), rgba(99, 102, 241, 0.9));
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.glass-button-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.glass-button-secondary {
  padding: 10px 20px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.8);
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.2s ease;
}

.glass-button-secondary:hover {
  background: rgba(255, 255, 255, 0.12);
  color: white;
}

/* 标签选择器样式 */
.tag-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  background: rgba(99, 102, 241, 0.2);
  border: 1px solid rgba(99, 102, 241, 0.4);
  border-radius: 16px;
  font-size: 13px;
  color: white;
  transition: all 0.2s ease;
}

.tag-badge:hover {
  background: rgba(99, 102, 241, 0.3);
}

.tag-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 4px;
  background: rgba(30, 30, 40, 0.98);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 12px;
  max-height: 200px;
  overflow-y: auto;
  z-index: 50;
  backdrop-filter: blur(12px);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.tag-option {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  cursor: pointer;
  transition: all 0.15s ease;
  color: rgba(255, 255, 255, 0.9);
  font-size: 14px;
}

.tag-option:hover {
  background: rgba(255, 255, 255, 0.08);
}

.tag-option:first-child {
  border-radius: 11px 11px 0 0;
}

.tag-option:last-child {
  border-radius: 0 0 11px 11px;
}

.tag-option:only-child {
  border-radius: 11px;
}

.tag-color-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-right: 10px;
  flex-shrink: 0;
}

.tag-no-result {
  padding: 12px 14px;
  color: rgba(255, 255, 255, 0.5);
  font-size: 13px;
  text-align: center;
}
</style>