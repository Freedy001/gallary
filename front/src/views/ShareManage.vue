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
                    class="rounded-lg px-3 py-1.5 text-xs font-medium text-primary-400/80 hover:text-primary-300 hover:bg-primary-500/10 transition-colors duration-300 ring-1 ring-transparent hover:ring-primary-500/20"
                    @click="extendShare(share)"
                >
                  修改有效期
                </button>
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

      <!-- 延期对话框 -->
      <Modal
        v-model="extendDialogVisible"
        size="md"
        title="修改过期时间"
        @close="closeExtendDialog"
      >
        <div class="space-y-6">
          <!-- 当前状态 -->
          <div class="p-4 rounded-xl bg-white/5 border border-white/10 flex items-center justify-between">
            <div class="flex items-center gap-3">
              <ClockIcon class="h-5 w-5 text-gray-400" />
              <div>
                <p class="text-xs text-gray-500 mb-0.5">当前过期时间</p>
                <p class="text-sm font-medium text-white">
                  {{ currentShare?.expire_at ? formatDate(currentShare.expire_at) : '永久有效' }}
                </p>
              </div>
            </div>
            <div
              :class="[
                'px-2.5 py-1 rounded-full text-xs font-medium border',
                currentShare && isExpired(currentShare)
                  ? 'bg-red-500/10 border-red-500/20 text-red-400'
                  : 'bg-green-500/10 border-green-500/20 text-green-400'
              ]"
            >
              {{ currentShare && isExpired(currentShare) ? '已过期' : '生效中' }}
            </div>
          </div>

          <!-- 快捷选项 -->
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-3">设置有效期</label>
            <div class="grid grid-cols-3 gap-3">
              <button
                v-for="option in extendOptions"
                :key="option.value"
                :class="[
                  'relative px-4 py-3 rounded-xl border text-sm font-medium transition-all duration-200',
                  extendOption === option.value
                    ? option.value === -1
                      ? 'bg-red-500/20 border-red-500 text-red-400'
                      : 'bg-primary-500/20 border-primary-500 text-primary-400'
                    :  option.value === -1?
                    'bg-white/5 border-white/10 text-red-400 hover:bg-white/10 hover:border-white/20':
                     'bg-white/5 border-white/10 text-gray-400 hover:bg-white/10 hover:border-white/20'
                ]"
                type="button"
                @click="handleExtendOptionChange(option.value)"
              >
                {{ option.label }}
              </button>
            </div>
          </div>

          <!-- 自定义日期 -->
          <div v-if="extendOption === 'custom'" class="animate-fade-in-down">
            <label class="block text-sm font-medium text-gray-300 mb-2">选择具体日期</label>
            <div class="relative group">
              <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <CalendarIcon class="h-5 w-5 text-gray-500 group-focus-within:text-primary-500 transition-colors" />
              </div>
              <input
                v-model="customExpireDate"
                :min="minDate"
                class="w-full rounded-xl bg-white/5 border border-white/10 pl-10 pr-4 py-2.5 text-sm text-white focus:outline-none focus:ring-1 focus:ring-primary-500 focus:border-primary-500 transition-all"
                type="datetime-local"
              />
            </div>
            <p class="text-xs text-gray-500 mt-2 ml-1">
              设置的过期时间必须晚于当前时间
            </p>
          </div>
        </div>

        <template #footer>
          <div class="flex justify-end gap-3">
            <button
              class="px-5 py-2.5 rounded-xl border border-white/10 text-gray-400 hover:bg-white/5 hover:text-white transition-colors"
              @click="closeExtendDialog"
            >
              取消
            </button>
            <button
              :disabled="!isValid"
              class="px-5 py-2.5 rounded-xl bg-primary-500 text-white hover:bg-primary-600 shadow-[0_0_15px_rgba(139,92,246,0.3)] hover:shadow-[0_0_20px_rgba(139,92,246,0.5)] transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none"
              @click="confirmExtend"
            >
              确认修改
            </button>
          </div>
        </template>
      </Modal>
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {computed, onMounted, ref} from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Modal from '@/components/widgets/common/Modal.vue'
import {shareApi} from '@/api/share'
import {useDialogStore} from '@/stores/dialog'
import type {Share} from '@/types'
import {
  ArrowDownTrayIcon,
  CalendarIcon,
  ClipboardDocumentIcon,
  ClockIcon,
  EyeIcon,
  LinkIcon,
  LockClosedIcon,
  ShareIcon
} from '@heroicons/vue/24/outline'

const loading = ref(false)
const shares = ref<Share[]>([])
const currentPage = ref(1)
const totalPages = ref(1)
const dialogStore = useDialogStore()

// 延期相关
const extendDialogVisible = ref(false)
const currentShare = ref<Share | null>(null)
const extendOption = ref<string | number>('')
const customExpireDate = ref('')

const extendOptions = [
  {label: '立即过期', value: -1},
  {label: '1天后', value: 1},
  {label: '7天后', value: 7},
  {label: '30天后', value: 30},
  {label: '永久有效', value: 0},
  {label: '自定义', value: 'custom'}
]

const minDate = computed(() => {
  const now = new Date()
  now.setMinutes(now.getMinutes() - now.getTimezoneOffset())
  return now.toISOString().slice(0, 16)
})

const isValid = computed(() => {
  if (extendOption.value === 0 || extendOption.value === -1) return true
  if (extendOption.value === 'custom') {
    return customExpireDate.value && new Date(customExpireDate.value) > new Date()
  }
  return !!extendOption.value
})

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

// 延期相关函数
function extendShare(share: Share) {
  currentShare.value = share
  extendOption.value = ''
  customExpireDate.value = ''
  extendDialogVisible.value = true
}

function closeExtendDialog() {
  extendDialogVisible.value = false
  currentShare.value = null
  extendOption.value = ''
  customExpireDate.value = ''
}

function handleExtendOptionChange(value: string | number) {
  extendOption.value = value
  if (value === 'custom') {
    customExpireDate.value = ''
  } else if (value === 0) {
    // 永久有效
    customExpireDate.value = ''
  } else if (value === -1) {
    // 立即过期
    customExpireDate.value = ''
  } else if (typeof value === 'number' && value > 0) {
    // 计算未来日期
    const futureDate = new Date()
    futureDate.setDate(futureDate.getDate() + value)
    // 格式化为 datetime-local 所需的格式
    const year = futureDate.getFullYear()
    const month = String(futureDate.getMonth() + 1).padStart(2, '0')
    const day = String(futureDate.getDate()).padStart(2, '0')
    const hours = String(futureDate.getHours()).padStart(2, '0')
    const minutes = String(futureDate.getMinutes()).padStart(2, '0')
    customExpireDate.value = `${year}-${month}-${day}T${hours}:${minutes}`
  }
}

async function confirmExtend() {
  if (!currentShare.value) return

  let expireAt: string | null = null

  if (extendOption.value === 0) {
    // 永久有效
    expireAt = null
  } else if (extendOption.value === -1) {
    // 立即过期 - 设置为当前时间之前
    expireAt = new Date().toISOString()
  } else if (extendOption.value === 'custom') {
    if (!customExpireDate.value) {
      dialogStore.alert({title: '错误', message: '请选择过期时间', type: 'error'})
      return
    }
    const selectedDate = new Date(customExpireDate.value)
    if (selectedDate <= new Date()) {
      dialogStore.alert({title: '错误', message: '过期时间必须大于当前时间', type: 'error'})
      return
    }
    expireAt = selectedDate.toISOString()
  } else if (typeof extendOption.value === 'number') {
    // 计算未来日期
    const futureDate = new Date()
    futureDate.setDate(futureDate.getDate() + extendOption.value)
    expireAt = futureDate.toISOString()
  } else {
    dialogStore.alert({title: '错误', message: '请选择延期时长', type: 'error'})
    return
  }

  try {
    await shareApi.update(currentShare.value.id, {expire_at: expireAt})
    dialogStore.alert({title: '成功', message: '有效期已更新', type: 'success'})
    closeExtendDialog()
    fetchShares(currentPage.value)
  } catch (error) {
    console.error('Failed to update share:', error)
    dialogStore.alert({title: '错误', message: '更新失败', type: 'error'})
  }
}

onMounted(() => {
  fetchShares()
})
</script>
