<template>
  <Teleport to="body">
    <Transition name="command-palette">
      <div
          v-if="uiStore.commandPaletteOpen"
          class="fixed inset-0 z-50 flex items-start justify-center bg-black/60 backdrop-blur-sm p-4 pt-[10vh]"
          @click.self="close"
          @keydown.esc="close"
      >
        <div class="w-full max-w-2xl overflow-hidden rounded-2xl border border-white/10 bg-[#0a0a0a]/90 shadow-[0_0_50px_-12px_rgba(0,0,0,0.8)] backdrop-blur-xl ring-1 ring-white/5" @click.stop>
          <!-- 搜索输入框 -->
          <div class="border-b border-white/5 px-5 py-5">
            <div class="flex items-center gap-4">
              <MagnifyingGlassIcon class="h-6 w-6 flex-shrink-0 text-primary-500 animate-pulse"/>
              <input
                  ref="searchInputRef"
                  v-model="searchQuery"
                  type="text"
                  placeholder="搜索影像记忆 / 日期 / 地点..."
                  class="flex-1 border-none bg-transparent text-lg text-white placeholder:text-gray-600 focus:outline-none font-light tracking-wide"
                  @keydown.down.prevent="selectNext"
                  @keydown.up.prevent="selectPrevious"
                  @keydown.enter="executeSearch"
              />
              <kbd class="rounded-md bg-white/10 px-2 py-1 text-xs font-mono text-gray-400 border border-white/5">ESC</kbd>
            </div>
          </div>

          <!-- 筛选选项 -->
          <div class="border-b border-white/5 px-5 py-4 bg-white/[0.02]">
            <div class="flex flex-wrap gap-2">
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

              <!-- 相机型号 -->
              <button
                  @click="toggleFilter('camera')"
                  :class="[
                  'flex items-center gap-1.5 rounded-full px-4 py-1.5 text-xs font-medium transition-all duration-300 border',
                  activeFilters.has('camera')
                    ? 'bg-primary-500/20 text-primary-300 border-primary-500/30 shadow-[0_0_10px_rgba(139,92,246,0.2)]'
                    : 'bg-white/5 text-gray-400 border-transparent hover:bg-white/10 hover:text-gray-200',
                ]"
              >
                <CameraIcon class="h-3.5 w-3.5"/>
                相机设备
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

          <!-- 筛选器详细配置 -->
          <div v-if="activeFilters.size > 0" class="border-b border-white/5 bg-black/20 px-5 py-6 animate-slide-in-top">
            <!-- 日期筛选 -->
            <div v-if="activeFilters.has('date')" class="mb-6 last:mb-0">
              <label class="mb-3 block text-sm font-medium text-gray-300">日期范围</label>
              <div class="flex items-center gap-3">
                <input
                    v-model="filters.start_date"
                    type="date"
                    class="flex-1 rounded-xl border border-white/10 bg-white/5 px-4 py-2.5 text-sm text-white transition-colors focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
                />
                <span class="text-sm font-medium text-gray-600">至</span>
                <input
                    v-model="filters.end_date"
                    type="date"
                    class="flex-1 rounded-xl border border-white/10 bg-white/5 px-4 py-2.5 text-sm text-white transition-colors focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
                />
              </div>
            </div>

            <!-- 相机筛选 -->
            <div v-if="activeFilters.has('camera')" class="mb-6 last:mb-0">
              <label class="mb-3 block text-sm font-medium text-gray-300">相机型号</label>
              <input
                  v-model="filters.camera_model"
                  type="text"
                  placeholder="例如: Canon EOS R5"
                  class="w-full rounded-xl border border-white/10 bg-white/5 px-4 py-2.5 text-sm text-white transition-colors placeholder:text-gray-600 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
              />
            </div>

            <!-- 位置筛选 -->
            <div v-if="activeFilters.has('location')" class="mb-6 last:mb-0">
              <label class="mb-3 block text-sm font-medium text-gray-300">位置名称</label>
              <input
                  v-model="filters.location"
                  type="text"
                  placeholder="例如: 北京"
                  class="w-full rounded-xl border border-white/10 bg-white/5 px-4 py-2.5 text-sm text-white transition-colors placeholder:text-gray-600 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
              />
            </div>

            <!-- 标签筛选 -->
            <div v-if="activeFilters.has('tags')" class="mb-6 last:mb-0">
              <label class="mb-3 block text-sm font-medium text-gray-300">标签</label>
              <input
                  v-model="filters.tags"
                  type="text"
                  placeholder="多个标签用逗号分隔"
                  class="w-full rounded-xl border border-white/10 bg-white/5 px-4 py-2.5 text-sm text-white transition-colors placeholder:text-gray-600 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
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
                  class="rounded-xl bg-primary-600 px-6 py-2 text-sm font-bold text-white shadow-[0_0_20px_rgba(124,58,237,0.4)] transition-all hover:bg-primary-500 hover:shadow-[0_0_30px_rgba(124,58,237,0.6)] active:scale-95"
              >
                搜索影像
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
import {imageApi} from "@/api/image.ts";

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
  location: '',
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
    location: '',
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
  if (filters.value.location) searchParams.location = filters.value.location
  if (filters.value.tags) searchParams.tags = filters.value.tags

  // 执行搜索
  try {
    await imageStore.refreshImages(async (page, size) => {
      searchParams.page = page
      searchParams.page_size = size
      return (await imageApi.search(searchParams)).data
    })

    close()

    // 确保在画廊页面
    if (router.currentRoute.value.path !== '/gallery') {
      await router.push('/gallery')
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
