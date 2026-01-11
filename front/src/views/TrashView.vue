<template>
  <AppLayout>
    <template #header>
      <div
          class="relative z-30 flex h-20 w-full items-center justify-between border-b border-white/5 bg-transparent px-8 transition-all duration-300 backdrop-blur-sm">
        <!-- 左侧区域 -->
        <div class="flex items-center gap-4">
          <!-- 正常模式：标题 -->
          <div v-if="!uiStore.isSelectionMode" class="flex items-center gap-3">
            <div
                class="flex h-10 w-10 items-center justify-center rounded-xl bg-white/5 text-primary-400 ring-1 ring-white/10">
              <TrashIcon class="h-5 w-5"/>
            </div>
            <div v-if="imageGridRef?.total" class="flex flex-col">
              <h1 class="text-lg font-medium text-white leading-tight">最近删除</h1>
              <span class="text-xs text-gray-500 font-mono mt-0.5">{{ imageGridRef.total }} 项资源</span>
            </div>
          </div>

          <!-- 选择模式：计数器 + 全选 (参考 TopBar) -->
          <div v-else class="flex items-center gap-6 animate-fade-in">
            <div class="flex items-center gap-3 rounded-lg bg-primary-500/10 px-4 py-2 border border-primary-500/20">
              <span class="h-2 w-2 rounded-full bg-primary-500 animate-pulse"></span>
              <span class="text-lg font-medium text-white">已选择 {{ selectedCount }} 项</span>
            </div>
            <button
                @click="toggleSelectAll"
                class="text-sm font-medium text-primary-400 hover:text-primary-300 hover:underline underline-offset-4"
            >
              {{ isAllSelected ? '取消全选' : '全选所有' }}
            </button>
          </div>
        </div>

        <!-- 右侧操作区域 -->
        <div class="flex items-center gap-6">
          <!-- 选择模式下的操作按钮 -->
          <div v-if="uiStore.isSelectionMode" class="flex items-center gap-4">
            <button
                :disabled="selectedCount === 0"
                @click="handleBatchRestore"
                class="flex items-center gap-2 rounded-xl bg-blue-500/10 border border-blue-500/20 px-5 py-2.5 text-sm font-medium text-blue-400 transition-all hover:bg-blue-500/20 hover:shadow-[0_0_15px_rgba(59,130,246,0.2)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-none"
            >
              <ArrowUturnLeftIcon class="h-4 w-4"/>
              <span>恢复 ({{ selectedCount }})</span>
            </button>

            <button
                :disabled="selectedCount === 0"
                @click="handleBatchPermanentDelete"
                class="flex items-center gap-2 rounded-xl bg-red-500/10 border border-red-500/20 px-5 py-2.5 text-sm font-medium text-red-400 transition-all hover:bg-red-500/20 hover:shadow-[0_0_15px_rgba(239,68,68,0.2)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-none"
            >
              <XMarkIcon class="h-4 w-4"/>
              <span>彻底删除 ({{ selectedCount }})</span>
            </button>

            <button
                @click="exitSelectionMode"
                class="rounded-xl border border-white/10 bg-white/5 px-6 py-2.5 text-sm font-medium text-white hover:bg-white/10 transition-colors"
            >
              完成
            </button>
          </div>

          <!-- 正常模式下的操作按钮 -->
          <div v-else-if="(imageGridRef?.images.length || 0 )> 0" class="flex items-center gap-4">
            <button
                @click="enterSelectionMode"
                class="rounded-xl px-4 py-2.5 text-sm font-medium text-gray-400 hover:bg-white/5 hover:text-white transition-colors"
            >
              选择
            </button>
          </div>
        </div>
      </div>
    </template>

    <template #default>
      <ImageGrid
          ref="imageGridRef"
          :fetcher="trashFetcher"
          mode="trash"
          @update:selected-count="selectedCount = $event"
      />
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useUIStore} from '@/stores/ui'
import {useDialogStore} from '@/stores/dialog'
import {imageApi} from '@/api/image'
import {dataSyncService} from '@/services/dataSync'
import type {Image, Pageable} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import ImageGrid from '@/components/gallery/ImageGrid.vue'
import {ArrowUturnLeftIcon, TrashIcon, XMarkIcon,} from '@heroicons/vue/24/outline'

const uiStore = useUIStore()
const dialogStore = useDialogStore()
const imageGridRef = ref<InstanceType<typeof ImageGrid> | null>(null)

// 本地状态
const selectedCount = ref(0)

// 回收站数据获取函数
const trashFetcher = async (page: number, size: number): Promise<Pageable<Image>> => {
  return (await imageApi.getDeletedList(page, size)).data
}

const isAllSelected = computed(() => {
  const grid = imageGridRef.value
  if (!grid) return false
  const validImages = grid.images.filter(img => img !== null) as Image[]
  return validImages.length > 0 && selectedCount.value === validImages.length
})

function toggleSelectAll() {
  const grid = imageGridRef.value
  if (!grid) return

  if (isAllSelected.value) {
    grid.clearSelection()
  } else {
    grid.selectAll()
  }
}

function enterSelectionMode() {
  uiStore.setSelectionMode(true)
}

function exitSelectionMode() {
  uiStore.setSelectionMode(false)
  imageGridRef.value?.clearSelection()
}

async function handleBatchRestore() {
  const ids = imageGridRef.value?.selectedIds
  if (!ids || ids.length === 0) return

  try {
    await imageApi.restoreImages(ids)
    // 刷新列表
    await imageGridRef.value?.refresh()

    // 发送数据同步事件，通知其他组件刷新
    dataSyncService.emit('images:restored', {ids, source: 'trash'})

    // 如果没有图片了，退出选择模式
    if ((imageGridRef?.value?.total || 0) === 0) {
      uiStore.setSelectionMode(false)
    }
  } catch (err) {
    console.error('批量恢复图片失败', err)
  }
}

async function handleBatchPermanentDelete() {
  const ids = imageGridRef.value?.selectedIds
  if (!ids || ids.length === 0) return

  const confirmed = await dialogStore.confirm({
    title: '确认彻底删除',
    message: `确定要彻底删除 ${ids.length} 张图片吗？此操作不可恢复。`,
    type: 'error',
    confirmText: '彻底删除'
  })

  if (!confirmed) return

  try {
    await imageApi.permanentlyDelete(ids)
    // 刷新列表
    await imageGridRef.value?.refresh()

    // 如果没有图片了，退出选择模式
    if ((imageGridRef?.value?.total || 0) === 0) {
      uiStore.setSelectionMode(false)
    }
  } catch (err) {
    console.error('批量彻底删除图片失败', err)
  }
}
</script>
