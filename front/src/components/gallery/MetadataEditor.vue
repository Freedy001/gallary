<template>
  <Modal
    :model-value="modelValue"
    :title="title"
    size="lg"
    @update:model-value="close"
  >
    <form @submit.prevent="save" class="space-y-4">
      <!-- Single Image Fields -->
      <div v-if="isSingleMode">
        <label class="block text-sm font-medium text-gray-800 ">文件名</label>
        <input
          v-model="form.original_name"
          type="text"
          class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white px-3 py-2 border"
        />
      </div>

      <!-- Common Fields -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="block text-sm font-medium text-gray-800">地点名称</label>
          <input
            v-model="form.location_name"
            type="text"
            class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white px-3 py-2 border"
            placeholder="输入地点名称"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-800">经纬度</label>
          <div class="flex space-x-2">
            <input
              v-model.number="form.latitude"
              type="number"
              step="any"
              class="mt-1 block w-1/2 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white px-3 py-2 border"
              placeholder="纬度"
            />
            <input
              v-model.number="form.longitude"
              type="number"
              step="any"
              class="mt-1 block w-1/2 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white px-3 py-2 border"
              placeholder="经度"
            />
          </div>
        </div>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-800">标签 (逗号分隔)</label>
        <input
          v-model="tagsInput"
          type="text"
          class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white px-3 py-2 border"
          placeholder="风景, 2023"
        />
      </div>

      <!-- Metadata Key-Value Pairs -->
      <div>
        <div class="flex justify-between items-center mb-2">
          <label class="block text-sm font-medium text-gray-800">扩展元数据</label>
          <button
            type="button"
            @click="addMetadataField"
            class="text-sm text-blue-600 hover:text-blue-500"
          >
            + 添加字段
          </button>
        </div>
        <div v-for="(item, index) in form.metadata" :key="index" class="flex space-x-2 mb-2">
            <input
              v-model="item.key"
              type="text"
              class="block w-1/3 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white px-3 py-2 border"
              placeholder="键名"
            />
            <input
              v-model="item.value"
              type="text"
              class="block w-1/2 rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white px-3 py-2 border"
              placeholder="键值"
            />
            <button
                type="button"
                @click="removeMetadataField(index)"
                class="text-red-600 hover:text-red-500"
            >
                <XMarkIcon class="h-5 w-5" />
            </button>
        </div>
      </div>

    </form>
    <template #footer>
      <div class="flex justify-end space-x-3">
        <button
          type="button"
          class="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200 dark:hover:bg-gray-600"
          @click="close"
        >
          取消
        </button>
        <button
          type="button"
          class="rounded-md border border-transparent bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          @click="save"
          :disabled="loading"
        >
          {{ loading ? '保存中...' : '保存' }}
        </button>
      </div>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Modal from '@/components/common/Modal.vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import type { Image, UpdateMetadataRequest, MetadataUpdate } from '@/types'
import { imageApi } from '@/api/image'

const props = defineProps<{
  modelValue: boolean
  imageIds: number[]
  initialData?: Image | null // For single edit pre-fill
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved'): void
}>()

const loading = ref(false)
const tagsInput = ref('')

interface FormState {
    original_name?: string
    location_name?: string
    latitude?: number
    longitude?: number
    metadata: MetadataUpdate[]
}

const form = ref<FormState>({
    metadata: []
})

const isSingleMode = computed(() => props.imageIds.length === 1)
const title = computed(() => isSingleMode.value ? '编辑图片元数据' : `批量编辑 ${props.imageIds.length} 张图片`)

// Reset form when opening
watch(() => props.modelValue, (val) => {
    if (val) {
        if (isSingleMode.value && props.initialData) {
            // Pre-fill for single image
            form.value = {
                original_name: props.initialData.original_name,
                location_name: props.initialData.location_name || null,
                latitude: props.initialData.latitude || null,
                longitude: props.initialData.longitude || null,
                metadata: props.initialData.metadata?.map(m => ({
                    key: m.meta_key,
                    value: m.meta_value,
                    value_type: m.value_type
                })) || []
            }
            tagsInput.value = props.initialData.tags?.map(t => t.name).join(', ') || ''
        } else {
            // Empty for batch edit
            form.value = {
                metadata: []
            }
            tagsInput.value = ''
        }
    }
})

const addMetadataField = () => {
    form.value.metadata.push({ key: '', value: '', value_type: 'string' })
}

const removeMetadataField = (index: number) => {
    form.value.metadata.splice(index, 1)
}

const close = () => {
    emit('update:modelValue', false)
}

const save = async () => {
    try {
        loading.value = true

        const data: UpdateMetadataRequest = {
            image_ids: props.imageIds,
            ...form.value,
            tags: tagsInput.value.split(/[,，]/).map(t => t.trim()).filter(t => t),
        }

        // Filter out empty metadata
        data.metadata = data.metadata?.filter(m => m.key)

        await imageApi.updateMetadata(data)
        emit('saved')
        close()
    } catch (e) {
        console.error('Failed to update metadata', e)
        // Handle error (could add a toast here if available)
    } finally {
        loading.value = false
    }
}
</script>
