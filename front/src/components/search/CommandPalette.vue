<template>
  <Teleport to="body">
    <Transition name="command-palette">
      <div
          v-if="uiStore.commandPaletteOpen"
          class="fixed inset-0 z-50 flex items-start justify-center bg-black/10 p-4 pt-[10vh]"
          @click.self="close"
          @keydown.esc="close"
      >
        <div class="w-full max-w-2xl overflow-hidden rounded-xl bg-white shadow-xl ring-1 ring-gray-900/5" @click.stop>
          <!-- 搜索输入框 -->
          <div class="border-b border-gray-200 px-5 py-4">
            <div class="flex items-center gap-3">
              <MagnifyingGlassIcon class="h-5 w-5 flex-shrink-0 text-gray-400"/>
              <input
                  ref="searchInputRef"
                  v-model="searchQuery"
                  type="text"
                  placeholder="搜索文件名、日期、相机型号、位置..."
                  class="flex-1 border-none bg-transparent text-base focus:outline-none"
                  @keydown.down.prevent="selectNext"
                  @keydown.up.prevent="selectPrevious"
                  @keydown.enter="executeSearch"
              />
              <kbd class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-500">ESC</kbd>
            </div>
          </div>

          <!-- 筛选选项 -->
          <div class="border-b border-gray-200 px-5 py-4">
            <div class="flex flex-wrap gap-2">
              <!-- 日期范围 -->
              <button
                  @click="toggleFilter('date')"
                  :class="[
                  'flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium transition-colors',
                  activeFilters.has('date')
                    ? 'bg-blue-100 text-blue-700'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200',
                ]"
              >
                <CalendarIcon class="h-3 w-3"/>
                日期范围
              </button>

              <!-- 相机型号 -->
              <button
                  @click="toggleFilter('camera')"
                  :class="[
                  'flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium transition-colors',
                  activeFilters.has('camera')
                    ? 'bg-blue-100 text-blue-700'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200',
                ]"
              >
                <CameraIcon class="h-3 w-3"/>
                相机
              </button>

              <!-- GPS位置 -->
              <button
                  @click="toggleFilter('location')"
                  :class="[
                  'flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium transition-colors',
                  activeFilters.has('location')
                    ? 'bg-blue-100 text-blue-700'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200',
                ]"
              >
                <MapPinIcon class="h-3 w-3"/>
                位置
              </button>

              <!-- 标签 -->
              <button
                  @click="toggleFilter('tags')"
                  :class="[
                  'flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium transition-colors',
                  activeFilters.has('tags')
                    ? 'bg-blue-100 text-blue-700'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200',
                ]"
              >
                <TagIcon class="h-3 w-3"/>
                标签
              </button>
            </div>
          </div>

          <!-- 筛选器详细配置 -->
          <div v-if="activeFilters.size > 0" class="border-b border-gray-200 bg-gray-50 px-5 py-5">
            <!-- 日期筛选 -->
            <div v-if="activeFilters.has('date')" class="mb-5 last:mb-0">
              <label class="mb-3 block text-sm font-semibold text-gray-900">日期范围</label>
              <div class="flex items-center gap-3">
                <input
                    v-model="filters.start_date"
                    type="date"
                    class="flex-1 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm transition-colors focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                />
                <span class="text-sm font-medium text-gray-500">至</span>
                <input
                    v-model="filters.end_date"
                    type="date"
                    class="flex-1 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm transition-colors focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                />
              </div>
            </div>

            <!-- 相机筛选 -->
            <div v-if="activeFilters.has('camera')" class="mb-5 last:mb-0">
              <label class="mb-3 block text-sm font-semibold text-gray-900">相机型号</label>
              <input
                  v-model="filters.camera_model"
                  type="text"
                  placeholder="例如: Canon EOS R5"
                  class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm transition-colors placeholder:text-gray-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
            </div>

            <!-- 位置筛选 -->
            <div v-if="activeFilters.has('location')" class="mb-5 last:mb-0">
              <label class="mb-3 block text-sm font-semibold text-gray-900">位置名称</label>
              <input
                  v-model="filters.location_name"
                  type="text"
                  placeholder="例如: 北京"
                  class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm transition-colors placeholder:text-gray-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
            </div>

            <!-- 标签筛选 -->
            <div v-if="activeFilters.has('tags')" class="mb-5 last:mb-0">
              <label class="mb-3 block text-sm font-semibold text-gray-900">标签</label>
              <input
                  v-model="filters.tags"
                  type="text"
                  placeholder="多个标签用逗号分隔"
                  class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm transition-colors placeholder:text-gray-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex items-center justify-between bg-white px-5 py-4">
            <button
                @click="clearFilters"
                class="text-sm font-medium text-gray-600 transition-colors hover:text-gray-900"
            >
              清除筛选
            </button>

            <div class="flex gap-3">
              <button
                  @click="close"
                  class="rounded-lg border border-gray-300 bg-white px-5 py-2 text-sm font-medium text-gray-700 transition-all hover:bg-gray-50 hover:border-gray-400"
              >
                取消
              </button>
              <button
                  @click="executeSearch"
                  class="rounded-lg bg-blue-600 px-5 py-2 text-sm font-medium text-white shadow-sm transition-all hover:bg-blue-700 hover:shadow-md active:scale-95"
              >
                搜索
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import {ref, watch, onMounted, onUnmounted, nextTick} from 'vue'
import {useRouter} from 'vue-router'
import {useUIStore} from '@/stores/ui'
import {useImageStore} from '@/stores/image'
import {
  MagnifyingGlassIcon,
  CalendarIcon,
  CameraIcon,
  MapPinIcon,
  TagIcon,
} from '@heroicons/vue/24/outline'
import type {SearchParams} from '@/types'

const router = useRouter()
const uiStore = useUIStore()
const imageStore = useImageStore()

const searchInputRef = ref<HTMLInputElement>()
const searchQuery = ref('')
const selectedIndex = ref(0)

const activeFilters = ref(new Set<string>())
const filters = ref<Partial<SearchParams>>({
  keyword: '',
  start_date: '',
  end_date: '',
  camera_model: '',
  location_name: '',
  tags: '',
})

// 监听命令面板打开，自动聚焦输入框
watch(() => uiStore.commandPaletteOpen, (isOpen) => {
  if (isOpen) {
    nextTick(() => {
      searchInputRef.value?.focus()
    })
  }
})

// 监听搜索输入
watch(searchQuery, (value) => {
  filters.value.keyword = value
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
  filters.value = {
    keyword: searchQuery.value,
    start_date: '',
    end_date: '',
    camera_model: '',
    location_name: '',
    tags: '',
  }
}

async function executeSearch() {
  // 构建搜索参数
  const searchParams: SearchParams = {}

  if (filters.value.keyword) searchParams.keyword = filters.value.keyword
  if (filters.value.start_date) searchParams.start_date = filters.value.start_date
  if (filters.value.end_date) searchParams.end_date = filters.value.end_date
  if (filters.value.camera_model) searchParams.camera_model = filters.value.camera_model
  if (filters.value.location_name) searchParams.location_name = filters.value.location_name
  if (filters.value.tags) searchParams.tags = filters.value.tags

  // 执行搜索
  try {
    await imageStore.searchImages(searchParams)
    close()

    // 确保在画廊页面
    if (router.currentRoute.value.path !== '/gallery') {
      router.push('/gallery')
    }
  } catch (error) {
    console.error('Search failed:', error)
  }
}

function selectNext() {
  selectedIndex.value = Math.min(selectedIndex.value + 1, 10)
}

function selectPrevious() {
  selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
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
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>
