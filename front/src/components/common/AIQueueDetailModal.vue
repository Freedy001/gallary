<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
          v-if="visible"
          class="fixed inset-0 z-50 flex items-center justify-center"
          @click.self="close"
      >
        <!-- 遮罩层 -->
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm"></div>

        <!-- 弹窗内容 -->
        <div class="relative bg-gray-900 rounded-xl shadow-2xl w-full max-w-2xl max-h-[80vh] flex flex-col border border-gray-700/50">
          <!-- 头部 -->
          <div class="flex items-center justify-between px-6 py-4 border-b border-gray-700/50">
            <div class="flex items-center gap-3">
              <div class="w-2 h-2 rounded-full" :class="getStatusDotClass()"></div>
              <h3 class="text-lg font-medium text-gray-100">
                {{ getQueueDisplayName(queue) }}
              </h3>
            </div>
            <button
                @click="close"
                class="text-gray-400 hover:text-gray-200 transition-colors"
            >
              <XMarkIcon class="h-5 w-5" />
            </button>
          </div>

          <!-- 统计信息 -->
          <div class="px-6 py-4 border-b border-gray-700/50 bg-gray-800/30">
            <div class="grid grid-cols-3 gap-4">
              <div class="text-center">
                <div class="text-2xl font-bold text-yellow-400">{{ queue.pending_count }}</div>
                <div class="text-xs text-gray-500">待处理</div>
              </div>
              <div class="text-center">
                <div class="text-2xl font-bold text-primary-400">{{ queue.processing_count }}</div>
                <div class="text-xs text-gray-500">处理中</div>
              </div>
              <div class="text-center">
                <div class="text-2xl font-bold text-red-400">{{ queue.failed_count }}</div>
                <div class="text-xs text-gray-500">失败</div>
              </div>
            </div>
          </div>

          <!-- 操作栏 -->
          <div class="px-6 py-3 border-b border-gray-700/50 flex items-center justify-between">
            <span class="text-sm text-gray-400">
              失败图片列表
            </span>
            <button
                v-if="queue.failed_count > 0"
                @click="retryAll"
                :disabled="aiStore.loading"
                class="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-primary-600 hover:bg-primary-500 text-white rounded-lg transition-colors disabled:opacity-50"
            >
              <ArrowPathIcon class="h-4 w-4" />
              全部重试
            </button>
          </div>

          <!-- 失败图片列表 -->
          <div class="flex-1 overflow-y-auto px-6 py-4">
            <div v-if="loading" class="flex items-center justify-center py-8">
              <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
            </div>

            <div v-else-if="failedImages.length === 0" class="flex flex-col items-center justify-center py-8 text-gray-500">
              <CheckCircleIcon class="h-12 w-12 mb-2 text-green-500/50" />
              <span>暂无失败图片</span>
            </div>

            <div v-else class="space-y-3">
              <div
                  v-for="item in failedImages"
                  :key="item.id"
                  class="flex items-start gap-4 p-3 bg-gray-800/50 rounded-lg border border-gray-700/50"
              >
                <!-- 缩略图 -->
                <div class="w-16 h-16 flex-shrink-0 rounded-lg overflow-hidden bg-gray-700">
                  <img
                      v-if="item.thumbnail"
                      :src="getThumbnailUrl(item.thumbnail)"
                      :alt="item.image_path"
                      class="w-full h-full object-cover"
                  />
                  <div v-else class="w-full h-full flex items-center justify-center text-gray-500">
                    <PhotoIcon class="h-8 w-8" />
                  </div>
                </div>

                <!-- 信息 -->
                <div class="flex-1 min-w-0">
                  <div class="text-sm text-gray-300 truncate mb-1">
                    {{ getFileName(item.image_path) }}
                  </div>
                  <div v-if="item.error" class="text-xs text-red-400 line-clamp-2">
                    {{ item.error }}
                  </div>
                  <div class="text-xs text-gray-500 mt-1">
                    {{ formatTime(item.created_at) }}
                  </div>
                </div>

                <!-- 操作按钮 -->
                <div class="flex-shrink-0 flex items-center gap-2">
                  <button
                      @click="retryImage(item.id)"
                      :disabled="aiStore.loading"
                      class="p-1.5 text-primary-400 hover:text-primary-300 hover:bg-primary-500/10 rounded transition-colors disabled:opacity-50"
                      title="重试"
                  >
                    <ArrowPathIcon class="h-4 w-4" />
                  </button>
                  <button
                      @click="ignoreImage(item.id)"
                      :disabled="aiStore.loading"
                      class="p-1.5 text-gray-400 hover:text-gray-300 hover:bg-gray-500/10 rounded transition-colors disabled:opacity-50"
                      title="忽略"
                  >
                    <XMarkIcon class="h-4 w-4" />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 分页 -->
          <div v-if="totalPages > 1" class="px-6 py-3 border-t border-gray-700/50 flex items-center justify-center gap-2">
            <button
                @click="prevPage"
                :disabled="currentPage <= 1"
                class="px-3 py-1 text-sm text-gray-400 hover:text-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              上一页
            </button>
            <span class="text-sm text-gray-500">
              {{ currentPage }} / {{ totalPages }}
            </span>
            <button
                @click="nextPage"
                :disabled="currentPage >= totalPages"
                class="px-3 py-1 text-sm text-gray-400 hover:text-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              下一页
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { XMarkIcon, ArrowPathIcon, CheckCircleIcon, PhotoIcon } from '@heroicons/vue/24/outline'
import { useAIStore } from '@/stores/ai'
import type { AIQueueInfo, AITaskImageInfo } from '@/types/ai'
import { getQueueDisplayName } from '@/types/ai'

const props = defineProps<{
  queue: AIQueueInfo
  visible: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const aiStore = useAIStore()

const loading = ref(false)
const currentPage = ref(1)
const pageSize = 10

// 计算属性
const failedImages = computed((): AITaskImageInfo[] => {
  return aiStore.queueDetail?.failed_images || []
})

const totalFailed = computed(() => {
  return aiStore.queueDetail?.total_failed || 0
})

const totalPages = computed(() => {
  return Math.ceil(totalFailed.value / pageSize)
})

// 获取状态点样式
function getStatusDotClass(): string {
  if (props.queue.status === 'processing' || props.queue.processing_count > 0) {
    return 'bg-primary-500 animate-pulse'
  }
  if (props.queue.pending_count > 0) {
    return 'bg-yellow-500'
  }
  if (props.queue.failed_count > 0) {
    return 'bg-red-500'
  }
  return 'bg-green-500'
}

// 获取缩略图 URL
function getThumbnailUrl(path: string): string {
  return `/resouse/${path}/file`
}

// 获取文件名
function getFileName(path: string): string {
  return path.split('/').pop() || path
}

// 格式化时间
function formatTime(timeStr: string): string {
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 加载数据
async function loadData() {
  loading.value = true
  try {
    await aiStore.fetchQueueDetail(props.queue.id, currentPage.value, pageSize)
  } finally {
    loading.value = false
  }
}

// 重试单张图片
async function retryImage(taskImageId: number) {
  try {
    await aiStore.retryTaskImage(taskImageId)
  } catch (error) {
    console.error('重试失败:', error)
  }
}

// 忽略单张图片
async function ignoreImage(taskImageId: number) {
  try {
    await aiStore.ignoreTaskImage(taskImageId)
  } catch (error) {
    console.error('忽略失败:', error)
  }
}

// 全部重试
async function retryAll() {
  try {
    await aiStore.retryQueueFailedImages(props.queue.id)
  } catch (error) {
    console.error('全部重试失败:', error)
  }
}

// 分页
function prevPage() {
  if (currentPage.value > 1) {
    currentPage.value--
    loadData()
  }
}

function nextPage() {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
    loadData()
  }
}

// 关闭
function close() {
  aiStore.clearQueueDetail()
  emit('close')
}

// 监听可见性变化
watch(() => props.visible, (newVal) => {
  if (newVal) {
    currentPage.value = 1
    loadData()
  }
})

// 监听队列变化
watch(() => props.queue.id, () => {
  if (props.visible) {
    currentPage.value = 1
    loadData()
  }
})
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active > div:last-child,
.modal-leave-active > div:last-child {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from > div:last-child,
.modal-leave-to > div:last-child {
  transform: scale(0.95);
  opacity: 0;
}
</style>
