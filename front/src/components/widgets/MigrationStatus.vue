<template>
  <div
      ref="containerRef"
      class="group cursor-pointer hover:bg-white/5 -mx-2 px-2 py-2 rounded-lg transition-colors"
      @click="openModal"
  >
    <!-- 展开状态 -->
    <template v-if="!collapsed">
      <div class="flex items-center justify-between text-xs text-gray-400 mb-1.5">
        <div class="flex items-center gap-1.5 font-mono tracking-wider">
          <ArrowsRightLeftIcon :class="migrationStore.hasTasks ? 'text-blue-400' : ''" class="h-3.5 w-3.5"/>
          <span>存储迁移</span>
        </div>
        <ArrowRightIcon class="h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity"/>
      </div>

      <!-- 状态摘要 -->
      <div class="space-y-1.5">
        <div class="flex items-center justify-between text-xs">
          <span class="text-gray-300">
            <template v-if="migrationStore.runningCount > 0">
              正在迁移
            </template>
            <template v-else-if="migrationStore.pausedCount > 0">
              已暂停
            </template>
            <template v-else-if="hasFailedTasks">
              需关注
            </template>
            <template v-else>
              空闲
            </template>
          </span>

          <span class="tabular-nums flex items-center gap-2">
            <!-- 失败数 -->
            <span v-if="totalFailed > 0" class="text-red-400 font-medium flex items-center gap-1">
              {{ totalFailed }} 失败
            </span>
            <!-- 暂停数 -->
            <span v-if="migrationStore.pausedCount > 0" class="text-yellow-400 font-medium flex items-center gap-1">
              {{ migrationStore.pausedCount }} 暂停
            </span>
            <!-- 运行数 -->
            <span v-if="migrationStore.runningCount > 0" class="text-blue-300">
              {{ migrationStore.runningCount }} 任务
            </span>
          </span>
        </div>

        <!-- 总体进度条 -->
        <div v-if="migrationStore.runningCount > 0" class="h-1 bg-white/5 rounded-full overflow-hidden">
          <div
              :class="getProgressBarClass"
              :style="{ width: `${(migrationStore.overallProgress)}%` }"
              class="h-full rounded-full transition-all duration-500 ease-out"
          ></div>
        </div>
      </div>
    </template>

    <!-- 收起状态 -->
    <template v-else>
      <div class="flex flex-col items-center gap-1">
        <div class="relative">
          <ArrowsRightLeftIcon class="h-4 w-4 text-gray-500 group-hover:text-blue-400 transition-colors"/>
          <!-- 状态指示点 -->
          <span
              v-if="migrationStore.runningCount > 0"
              class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-blue-500 animate-pulse border border-gray-900"
          ></span>
          <span
              v-else-if="migrationStore.pausedCount > 0"
              class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-yellow-500 border border-gray-900"
          ></span>
          <span
              v-else-if="hasFailedTasks"
              class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-red-500 border border-gray-900"
          ></span>
        </div>

        <!-- 简单的数量指示 -->
        <span v-if="migrationStore.runningCount > 0" class="text-[10px] text-blue-400 font-bold">
          {{ migrationStore.overallProgress }}%
        </span>
      </div>
    </template>

    <!-- 详情弹窗 -->
    <MigrationDetailModal
        :trigger-rect="triggerRect"
        :visible="modalVisible"
        @close="modalVisible = false"
    />
  </div>
</template>

<script lang="ts" setup>
import {computed, ref} from 'vue'
import {ArrowRightIcon, ArrowsRightLeftIcon} from '@heroicons/vue/24/outline'
import {useMigrationStore} from '@/stores/migration'
import MigrationDetailModal from '@/components/widgets/migration/MigrationDetailModal.vue'

defineProps<{
  collapsed: boolean
}>()

const migrationStore = useMigrationStore()
const containerRef = ref<HTMLElement | null>(null)
const modalVisible = ref(false)
const triggerRect = ref<DOMRect | null>(null)

// 打开弹窗
function openModal() {
  // 获取点击时的触发区域位置
  if (containerRef.value) {
    triggerRect.value = containerRef.value.getBoundingClientRect()
    modalVisible.value = true
  }
}

// 计算属性
// 失败相关
const totalFailed = computed(() => migrationStore.tasks.reduce((sum, task) => sum + task.failed_files, 0))
const hasFailedTasks = computed(() => totalFailed.value > 0)

const getProgressBarClass = computed(() => {
  if (migrationStore.runningCount > 0) {
    return 'bg-blue-500 animate-pulse'
  }
  if (migrationStore.pausedCount > 0) {
    return 'bg-yellow-500'
  }
  return 'bg-gray-700'
})
</script>
