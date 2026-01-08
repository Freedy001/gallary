<template>
  <Modal v-model="isOpen" :title="isEditMode ? '编辑相册' : '新建相册'" size="sm">
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
          {{ loading ? (isEditMode ? '保存中...' : '创建中...') : (isEditMode ? '保存' : '创建') }}
        </button>
      </div>
    </form>
  </Modal>
</template>

<script setup lang="ts">
import {computed, reactive, ref, watch} from 'vue'
import {useAlbumStore} from '@/stores/album'
import Modal from '@/components/widgets/common/Modal.vue'
import type {Album} from '@/types'

const props = defineProps<{
  editMode?: boolean
  initialData?: Album | null
}>()

const isOpen = defineModel<boolean>({ default: false })
const emit = defineEmits<{
  created: []
  updated: []
}>()

const albumStore = useAlbumStore()
const loading = ref(false)

const isEditMode = computed(() => props.editMode && !!props.initialData)

const form = reactive({
  name: '',
  description: ''
})

// 重置表单或填充初始数据
watch([isOpen, () => props.initialData], ([val, data]) => {
  if (val) {
    if (isEditMode.value && data) {
      form.name = data.name
      form.description = data.description || ''
    } else {
      form.name = ''
      form.description = ''
    }
  }
})

async function handleSubmit() {
  if (!form.name.trim() || loading.value) return

  try {
    loading.value = true

    if (isEditMode.value && props.initialData) {
      await albumStore.updateAlbum(props.initialData.id, form.name.trim(), form.description.trim() || undefined)
      emit('updated')
    } else {
      await albumStore.createAlbum(form.name.trim(), form.description.trim() || undefined)
      emit('created')
    }

    isOpen.value = false
  } catch (err) {
    console.error(isEditMode.value ? '更新相册失败' : '创建相册失败', err)
  } finally {
    loading.value = false
  }
}
</script>
