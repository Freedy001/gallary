<template>
  <AppLayout>
    <template #header>
      <header
          class="sticky top-0 z-30 flex h-16 w-full items-center justify-between border-b border-white/5 bg-black/20 backdrop-blur-xl px-6 transition-all duration-300">
        <h1 class="text-xl font-bold tracking-wide text-white/90 font-display">分享管理</h1>
      </header>
    </template>

    <template #default>
      <div class="p-6 min-h-[calc(100vh-4rem)]">
        <!-- 加载状态 -->
        <div v-if="loading" class="flex h-64 items-center justify-center">
          <div
              class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent shadow-[0_0_15px_rgba(139,92,246,0.5)]"></div>
        </div>

        <!-- 空状态 -->
        <div v-else-if="shares.length === 0" class="flex h-64 flex-col items-center justify-center text-gray-500">
          <div class="rounded-2xl bg-white/5 p-4 mb-4 ring-1 ring-white/10">
            <ShareIcon class="h-8 w-8 text-gray-400"/>
          </div>
          <p class="text-sm text-gray-400 font-light tracking-wide">暂无分享记录</p>
        </div>

        <!-- 分享列表 -->
        <div v-else class="grid gap-6 sm:grid-cols-1 lg:grid-cols-2 xl:grid-cols-3">
          <div
              v-for="share in shares"
              :key="share.id"
              class="group relative flex flex-col overflow-hidden rounded-2xl bg-white/5 ring-1 ring-white/10 transition-all duration-500 hover:bg-white/[0.07] hover:ring-white/20 hover:scale-[1.01] hover:shadow-[0_0_30px_rgba(0,0,0,0.3)]"
          >
            <!-- 顶部装饰光效 -->
            <div
                class="absolute top-0 left-0 right-0 h-[1px] bg-gradient-to-r from-transparent via-white/20 to-transparent opacity-0 transition-opacity duration-500 group-hover:opacity-100"></div>

            <!-- 卡片头部 -->
            <div class="border-b border-white/5 p-5 bg-white/[0.02]">
              <div class="flex items-start justify-between gap-4">
                <div class="flex-1 min-w-0">
                  <h3 class="font-medium text-gray-100 truncate text-lg tracking-tight"
                      :title="share.title || '未命名分享'">
                    {{ share.title || '未命名分享' }}
                  </h3>
                  <p class="mt-1.5 text-xs text-gray-500 font-mono flex items-center gap-2">
                    <span class="w-1.5 h-1.5 rounded-full bg-gray-600"></span>
                    创建于 {{ formatDate(share.created_at) }}
                  </p>
                </div>
                <span
                    :class="[
                    'inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset backdrop-blur-sm',
                    isExpired(share)
                      ? 'bg-red-500/10 text-red-400 ring-red-500/20'
                      : 'bg-emerald-500/10 text-emerald-400 ring-emerald-500/20 shadow-[0_0_10px_rgba(52,211,153,0.1)]'
                  ]"
                >
                  {{ isExpired(share) ? '已过期' : '有效' }}
                </span>
              </div>
            </div>

            <!-- 卡片内容 -->
            <div class="p-5 flex-1 flex flex-col">
              <div class="space-y-3 text-sm text-gray-400 flex-1">
                <div class="grid grid-cols-2 gap-4 mb-4">
                  <div class="flex items-center gap-2.5 p-2 rounded-lg bg-white/[0.02] border border-white/5">
                    <EyeIcon class="h-4 w-4 text-gray-500"/>
                    <span class="font-mono text-gray-300">{{ share.view_count }}</span>
                    <span class="text-xs text-gray-600">浏览</span>
                  </div>
                  <div class="flex items-center gap-2.5 p-2 rounded-lg bg-white/[0.02] border border-white/5">
                    <ArrowDownTrayIcon class="h-4 w-4 text-gray-500"/>
                    <span class="font-mono text-gray-300">{{ share.download_count }}</span>
                    <span class="text-xs text-gray-600">下载</span>
                  </div>
                </div>

                <div class="flex items-center gap-2.5 text-xs">
                  <ClockIcon class="h-4 w-4 text-gray-600"/>
                  <span :class="isExpired(share) ? 'text-red-400/70' : 'text-gray-500'">{{
                      formatExpireDate(share.expire_at)
                    }}</span>
                </div>

                <div class="flex items-center gap-2.5 text-xs">
                  <LockClosedIcon class="h-4 w-4 text-gray-600"/>
                  <span v-if="share.password" class="text-gray-400 font-mono">{{ share.password }}</span>
                  <span v-else class="text-gray-600">无密码</span>
                  <button
                      v-if="share.password"
                      @click="copyPassword(share.password!)"
                      class="ml-1 p-1 rounded hover:bg-white/10 text-gray-500 hover:text-gray-300 transition-colors"
                      title="复制密码"
                  >
                    <ClipboardDocumentIcon class="h-3.5 w-3.5"/>
                  </button>
                </div>

                <div class="flex items-center gap-2.5 overflow-hidden group/link">
                  <LinkIcon class="h-4 w-4 text-gray-600 flex-shrink-0"/>
                  <a
                      :href="getShareLink(share.share_code)"
                      target="_blank"
                      class="text-primary-400/80 hover:text-primary-300 truncate font-mono text-xs transition-colors underline decoration-primary-500/30 underline-offset-4 hover:decoration-primary-400"
                  >
                    {{ getShareLink(share.share_code) }}
                  </a>
                </div>
              </div>

              <!-- 操作按钮 -->
              <div class="flex items-center justify-end gap-3 pt-5 mt-2">
                <button
                    @click="copyLink(share.share_code)"
                    class="rounded-lg px-3 py-1.5 text-xs font-medium text-gray-300 hover:text-white hover:bg-white/10 transition-colors duration-300 ring-1 ring-transparent hover:ring-white/10"
                >
                  复制链接
                </button>
                <button
                    @click="deleteShare(share)"
                    class="rounded-lg px-3 py-1.5 text-xs font-medium text-red-400/80 hover:text-red-300 hover:bg-red-500/10 transition-colors duration-300 ring-1 ring-transparent hover:ring-red-500/20"
                >
                  删除
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 分页 -->
        <div v-if="totalPages > 1" class="mt-12 flex justify-center">
          <nav class="flex items-center gap-3 p-1 rounded-xl bg-white/5 ring-1 ring-white/10 backdrop-blur-md">
            <button
                @click="changePage(currentPage - 1)"
                :disabled="currentPage === 1"
                class="rounded-lg px-4 py-2 text-sm text-gray-400 transition-all hover:text-white hover:bg-white/10 disabled:opacity-30 disabled:hover:bg-transparent"
            >
              上一页
            </button>
            <span class="text-sm font-mono text-primary-400 font-medium px-2">
              {{ currentPage }} <span class="text-gray-600">/</span> {{ totalPages }}
            </span>
            <button
                @click="changePage(currentPage + 1)"
                :disabled="currentPage === totalPages"
                class="rounded-lg px-4 py-2 text-sm text-gray-400 transition-all hover:text-white hover:bg-white/10 disabled:opacity-30 disabled:hover:bg-transparent"
            >
              下一页
            </button>
          </nav>
        </div>
      </div>
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {ref, onMounted} from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import {shareApi} from '@/api/share'
import {useDialogStore} from '@/stores/dialog'
import type {Share} from '@/types'
import {
  ShareIcon,
  EyeIcon,
  ArrowDownTrayIcon,
  ClockIcon,
  LinkIcon,
  LockClosedIcon,
  ClipboardDocumentIcon
} from '@heroicons/vue/24/outline'

const loading = ref(false)
const shares = ref<Share[]>([])
const currentPage = ref(1)
const totalPages = ref(1)
const dialogStore = useDialogStore()

async function fetchShares(page = 1) {
  loading.value = true
  try {
    const res = await shareApi.getList(page)
    // 标准化响应: { code, message, data: { list, total, page, page_size, total_pages } }
    shares.value = res.data.list || []
    totalPages.value = res.data.total_pages || 1
    currentPage.value = page
  } catch (error) {
    console.error('Failed to fetch shares:', error)
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: 'numeric',
    minute: 'numeric'
  })
}

function formatExpireDate(dateStr?: string) {
  if (!dateStr) return '永久有效'
  return '有效期至 ' + formatDate(dateStr)
}

function isExpired(share: Share) {
  if (!share.expire_at) return false
  return new Date(share.expire_at) < new Date()
}

function getShareLink(code: string) {
  return `${window.location.origin}/s/${code}`
}

async function copyLink(code: string) {
  const link = getShareLink(code)
  try {
    await navigator.clipboard.writeText(link)
    dialogStore.alert({title: '成功', message: '链接已复制到剪贴板', type: 'success'})
  } catch (err) {
    console.error('Failed to copy:', err)
    dialogStore.alert({title: '错误', message: '复制失败', type: 'error'})
  }
}

async function copyPassword(password: string) {
  try {
    await navigator.clipboard.writeText(password)
    dialogStore.alert({title: '成功', message: '密码已复制到剪贴板', type: 'success'})
  } catch (err) {
    console.error('Failed to copy:', err)
    dialogStore.alert({title: '错误', message: '复制失败', type: 'error'})
  }
}

async function deleteShare(share: Share) {
  const confirmed = await dialogStore.confirm({
    title: '确认删除',
    message: `确定要删除分享"${share.title || '未命名'}"吗？`,
    type: 'warning',
    confirmText: '删除'
  })

  if (!confirmed) return

  try {
    await shareApi.delete(share.id)
    fetchShares(currentPage.value)
  } catch (error) {
    console.error('Failed to delete share:', error)
    dialogStore.alert({title: '错误', message: '删除失败', type: 'error'})
  }
}

function changePage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    fetchShares(page)
  }
}

onMounted(() => {
  fetchShares()
})
</script>
