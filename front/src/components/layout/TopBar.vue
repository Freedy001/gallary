<template>
  <header class="relative z-30 flex h-20 w-full items-center justify-between border-b border-white/5 bg-transparent px-8 transition-all duration-300 backdrop-blur-sm">
    <!-- 左侧区域 -->
    <div v-if="!isSelectionMode" class="w-96">
      <slot name="left"></slot>
    </div>

    <!-- 选择模式下的左侧 -->
    <div v-else class="flex items-center gap-6 animate-fade-in">
      <div class="flex items-center gap-3 rounded-lg bg-primary-500/10 px-4 py-2 border border-primary-500/20">
        <span class="h-2 w-2 rounded-full bg-primary-500 animate-pulse"></span>
        <span class="text-lg font-medium text-white">已选择 {{ selectedCount }} 项</span>
      </div>
      <button
        @click="emit('selectAll')"
        class="text-sm font-medium text-primary-400 hover:text-primary-300 hover:underline underline-offset-4"
      >
        {{ isAllSelected ? '取消全选' : '全选所有' }}
      </button>
    </div>

    <!-- 右侧区域 -->
    <div class="flex items-center gap-6">
      <!-- 选择模式下的操作按钮 -->
      <div v-if="isSelectionMode" class="flex items-center gap-4">
        <slot name="selection-actions" />
        <button
          @click="emit('exitSelection')"
          class="rounded-xl border border-white/10 bg-white/5 px-6 py-2.5 text-sm font-medium text-white hover:bg-white/10 transition-colors"
        >
          完成
        </button>
      </div>

      <!-- 正常模式下的操作按钮 -->
      <div v-else class="flex items-center gap-4">
        <slot name="actions" />
      </div>

      <!-- 文件上传 input（可选） -->
      <input
        v-if="showUpload"
        ref="fileInputRef"
        type="file"
        multiple
        accept="image/*"
        webkitdirectory
        class="hidden"
        @change="handleFileSelect"
      />

      <!-- 分隔线（密度滑块或排序选择器可见时显示） -->
      <div v-if="showDensitySlider || showSortSelector" class="h-8 w-px bg-white/10" />

      <!-- 排序选择器（可选） -->
      <div v-if="showSortSelector" class="w-30">
        <BaseSelect
          :model-value="sortBy"
          :options="sortOptions"
          button-class="!py-2.5 !text-xm"
          placeholder="排序方式"
          @update:model-value="emit('sortChange', $event as SortBy)"
        />
      </div>

      <!-- 视图密度滑块（可选） -->
      <div v-if="showDensitySlider" class="flex items-center gap-3 group">
        <div class="flex items-center gap-2 px-2 py-1 rounded-lg group-hover:bg-white/5 transition-colors">
          <Squares2X2Icon class="h-4 w-4 text-gray-500 group-hover:text-gray-300" />
          <input
            type="range"
            min="1"
            :value="displayDensity"
            max="16"
            @input="handleDensityChange"
            class="w-24 cursor-pointer accent-white h-1 bg-white/10 rounded-full appearance-none hover:bg-white/20"
          />
          <Square3Stack3DIcon class="h-4 w-4 text-gray-500 group-hover:text-gray-300" />
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {Square3Stack3DIcon, Squares2X2Icon} from '@heroicons/vue/24/outline'
import {useDebounceFn} from '@vueuse/core'
import BaseSelect, {type SelectOption} from '@/components/common/BaseSelect.vue'
import type {SortBy} from '@/stores/ui'

const props = withDefaults(defineProps<{
  // 选择模式
  isSelectionMode?: boolean
  selectedCount?: number
  totalCount?: number
  // 密度滑块
  showDensitySlider?: boolean
  gridDensity?: number
  // 上传功能
  showUpload?: boolean
  uploadAlbumId?: number
  // 排序选择器
  showSortSelector?: boolean
  sortBy?: SortBy
}>(), {
  isSelectionMode: false,
  selectedCount: 0,
  totalCount: 0,
  showDensitySlider: true,
  gridDensity: 5,
  showUpload: false,
  showSortSelector: false,
  sortBy: 'taken_at',
})

const emit = defineEmits<{
  openSearch: []
  selectAll: []
  exitSelection: []
  densityChange: [value: number]
  filesSelected: [files: File[], albumId?: number]
  sortChange: [value: SortBy]
}>()

const fileInputRef = ref<HTMLInputElement>()

const isAllSelected = computed(() => {
  return props.totalCount > 0 && props.selectedCount === props.totalCount
})

const sortOptions: SelectOption[] = [
  { label: '拍摄时间', value: 'taken_at' },
  { label: '美学评分', value: 'ai_score' }
]

// 本地显示值，用于滑块即时响应
const displayDensity = ref(props.gridDensity)

// 同步 props 变化到本地显示值
watch(() => props.gridDensity, (val) => {
  displayDensity.value = val
})

// 防抖触发实际的密度变更，避免频繁触发布局重算
const debouncedEmit = useDebounceFn((value: number) => {
  emit('densityChange', value)
}, 150)

function handleDensityChange(event: Event) {
  const value = parseInt((event.target as HTMLInputElement).value)
  // 立即更新本地显示值（滑块位置即时响应）
  displayDensity.value = value
  // 防抖触发实际变更
  debouncedEmit(value)
}



function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files

  if (!files || files.length === 0) return

  // 过滤出图片文件（支持文件夹模式时可能包含非图片文件）
  const imageFiles = Array.from(files).filter(file => file.type.startsWith('image/'))

  if (imageFiles.length === 0) return

  emit('filesSelected', imageFiles, props.uploadAlbumId)

  // 清空 input，允许重复选择
  input.value = ''
}

// 暴露方法供父组件调用
function triggerUpload() {
  fileInputRef.value?.click()
}

defineExpose({ triggerUpload })
</script>
