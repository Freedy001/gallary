<template>
  <Modal v-model="isOpen" title="新建相册" size="sm">
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">相册名称</label>
        <input
          v-model="form.name"
          type="text"
          required
          placeholder="输入相册名称"
          class="w-full rounded-xl bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 transition-colors outline-none"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">相册描述（可选）</label>
        <textarea
          v-model="form.description"
          rows="3"
          placeholder="输入相册描述"
          class="w-full rounded-xl bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 transition-colors resize-none outline-none"
        />
      </div>

      <div class="flex justify-end gap-3 pt-4">
        <button
          type="button"
          @click="isOpen = false"
          class="px-5 py-2.5 rounded-xl border border-white/10 text-gray-400 hover:bg-white/5 transition-colors"
        >
          取消
        </button>
        <button
          type="submit"
          :disabled="loading || !form.name.trim()"
          class="px-5 py-2.5 rounded-xl bg-primary-500 text-white hover:bg-primary-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ loading ? '创建中...' : '创建' }}
        </button>
      </div>
    </form>
  </Modal>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { useAlbumStore } from '@/stores/album'
import Modal from '@/components/common/Modal.vue'

const isOpen = defineModel<boolean>({ default: false })
const emit = defineEmits<{
  created: []
}>()

const albumStore = useAlbumStore()
const loading = ref(false)

const form = reactive({
  name: '',
  description: ''
})

// 重置表单
watch(isOpen, (val) => {
  if (!val) {
    form.name = ''
    form.description = ''
  }
})

async function handleSubmit() {
  if (!form.name.trim() || loading.value) return

  try {
    loading.value = true
    await albumStore.createAlbum(form.name.trim(), form.description.trim() || undefined)
    isOpen.value = false
    emit('created')
  } catch (err) {
    console.error('创建相册失败', err)
  } finally {
    loading.value = false
  }
}
</script>
