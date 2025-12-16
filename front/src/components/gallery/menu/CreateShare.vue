<template>
  <Modal
    :model-value="isOpen"
    size="md"
    :closable="step === 'form'"
    @update:model-value="handleModalClose"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-white/10 backdrop-blur-sm">
          <ShareIcon v-if="step === 'form'" class="h-5 w-5 text-white" />
          <CheckCircleIcon v-else class="h-5 w-5 text-green-400" />
        </div>
        <h3 class="text-xl font-semibold text-white">
          {{ step === 'form' ? '创建分享' : '分享创建成功' }}
        </h3>
      </div>
    </template>

    <!-- 表单状态 -->
    <div v-if="step === 'form'">
      <p class="text-sm text-white/70">
        已选择 <span class="text-white font-medium">{{ selectedCount }}</span> 张图片进行分享
      </p>

      <form @submit.prevent="handleSubmit" class="mt-6 space-y-5">
        <!-- 标题输入 -->
        <div>
          <label class="block text-sm font-medium text-white/80 mb-2">标题</label>
          <input
            v-model="form.title"
            type="text"
            placeholder="给这次分享起个名字（可选）"
            class="glass-input"
          />
        </div>

        <!-- 描述输入 -->
        <div>
          <label class="block text-sm font-medium text-white/80 mb-2">描述</label>
          <textarea
            v-model="form.description"
            rows="3"
            placeholder="添加描述信息（可选）"
            class="glass-input resize-none"
          />
        </div>

        <!-- 有效期选择 -->
        <div>
          <label class="block text-sm font-medium text-white/80 mb-2">有效期</label>
          <div class="grid grid-cols-4 gap-2">
            <button
              v-for="option in expireOptions"
              :key="option.value"
              type="button"
              @click="form.expire_days = option.value"
              :class="[
                'glass-chip',
                form.expire_days === option.value ? 'glass-chip-active' : ''
              ]"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

        <!-- 访问密码 -->
        <div>
          <label class="block text-sm font-medium text-white/80 mb-2">访问密码</label>
          <div class="relative">
            <input
              v-model="form.password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="留空则无需密码"
              class="glass-input pr-24"
            />
            <button
              type="button"
              @click="showPassword = !showPassword"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-white/50 hover:text-white/80 transition-colors"
            >
              <EyeIcon v-if="!showPassword" class="h-5 w-5" />
              <EyeSlashIcon v-else class="h-5 w-5" />
            </button>
          </div>
          <button
            type="button"
            @click="generateRandomPassword"
            class="mt-2 text-sm text-white/60 hover:text-white/80 transition-colors flex items-center gap-1"
          >
            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            使用随机密码
          </button>
        </div>
      </form>
    </div>

    <!-- 成功状态 -->
    <div v-else>
      <div class="rounded-2xl bg-green-500/10 border border-green-500/20 p-4">
        <p class="text-sm text-green-300">
          分享链接已生成，可以复制发送给他人查看
        </p>
      </div>

      <div class="mt-6">
        <label class="block text-sm font-medium text-white/80 mb-2">分享链接</label>
        <div class="flex rounded-xl overflow-hidden border border-white/10">
          <input
            readonly
            type="text"
            :value="shareLink"
            class="flex-1 bg-white/5 px-4 py-3 text-white text-sm outline-none"
          />
          <button
            @click="copyLink"
            class="px-4 bg-white/10 text-white/80 hover:bg-white/20 hover:text-white transition-colors flex items-center gap-2"
          >
            <ClipboardDocumentIcon class="h-5 w-5" />
            {{ copied ? '已复制' : '复制' }}
          </button>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <template v-if="step === 'form'">
          <button
            type="button"
            class="glass-button-secondary"
            @click="closeModal"
          >
            取消
          </button>
          <button
            type="button"
            :disabled="loading"
            class="glass-button-primary"
            @click="handleSubmit"
          >
            <span v-if="loading" class="flex items-center gap-2">
              <svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              创建中...
            </span>
            <span v-else>创建分享</span>
          </button>
        </template>
        <template v-else>
          <button
            type="button"
            class="glass-button-primary"
            @click="closeModal"
          >
            完成
          </button>
        </template>
      </div>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import Modal from '@/components/common/Modal.vue'
import { shareApi } from '@/api/share.ts'
import { useDialogStore } from '@/stores/dialog'
import {
  ShareIcon,
  CheckCircleIcon,
  EyeIcon,
  EyeSlashIcon,
  ClipboardDocumentIcon,
} from '@heroicons/vue/24/outline'

const isOpen = defineModel<boolean>({ default: false })
const props = defineProps<{
  selectedCount: number
  selectedIds: number[]
}>()

const emit = defineEmits<{
  (e: 'created', shareCode: string): void
}>()

const loading = ref(false)
const step = ref<'form' | 'success'>('form')
const shareCode = ref('')
const shareLink = ref('')
const showPassword = ref(false)
const copied = ref(false)
const dialogStore = useDialogStore()

const expireOptions = [
  { label: '永久', value: 0 },
  { label: '1天', value: 1 },
  { label: '7天', value: 7 },
  { label: '30天', value: 30 },
]

const form = reactive({
  title: '',
  description: '',
  expire_days: 7,
  password: ''
})

function handleModalClose(value: boolean) {
  if (!value) {
    closeModal()
  }
}

function closeModal() {
  isOpen.value = false
  setTimeout(() => {
    step.value = 'form'
    form.title = ''
    form.description = ''
    form.expire_days = 7
    form.password = ''
    shareCode.value = ''
    shareLink.value = ''
    copied.value = false
  }, 300)
}

async function handleSubmit() {
  if (props.selectedIds.length === 0) return

  loading.value = true
  try {
    const res = await shareApi.create({
      image_ids: props.selectedIds,
      title: form.title,
      description: form.description,
      expire_days: form.expire_days,
      password: form.password || undefined
    })
    const code = res.data.share_code
    shareCode.value = code
    shareLink.value = `${window.location.origin}/s/${code}`
    step.value = 'success'
    emit('created', code)
  } catch (error) {
    console.error('Create share failed:', error)
    await dialogStore.alert({ title: '错误', message: '创建分享失败', type: 'error' })
  } finally {
    loading.value = false
  }
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(shareLink.value)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
    dialogStore.alert({ title: '错误', message: '复制失败', type: 'error' })
  }
}

function generateRandomPassword() {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let password = ''
  for (let i = 0; i < 8; i++) {
    password += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  form.password = password
  showPassword.value = true
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

.glass-chip {
  padding: 8px 12px;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.2s ease;
}

.glass-chip:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.glass-chip-active {
  background: rgba(59, 130, 246, 0.3);
  border-color: rgba(59, 130, 246, 0.5);
  color: white;
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
</style>
