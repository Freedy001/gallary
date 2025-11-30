<template>
  <div class="space-y-4 pt-4 border-t border-white/5">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">Endpoint</label>
        <input
            :value="endpoint"
            @input="$emit('update:endpoint', ($event.target as HTMLInputElement).value)"
            type="text"
            class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
            placeholder="oss-cn-hangzhou.aliyuncs.com"
        />
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">Bucket</label>
        <input
            :value="bucket"
            @input="$emit('update:bucket', ($event.target as HTMLInputElement).value)"
            type="text"
            class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
            placeholder="your-bucket-name"
        />
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">Access Key ID</label>
        <div class="relative">
          <input
              :value="accessKeyId"
              @input="$emit('update:accessKeyId', ($event.target as HTMLInputElement).value)"
              :type="showAccessKeyId ? 'text' : 'password'"
              class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 pr-12 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
              placeholder="Access Key ID"
          />
          <button
              type="button"
              @click="showAccessKeyId = !showAccessKeyId"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
          >
            <EyeIcon v-if="!showAccessKeyId" class="h-5 w-5" />
            <EyeSlashIcon v-else class="h-5 w-5" />
          </button>
        </div>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">Access Key Secret</label>
        <div class="relative">
          <input
              :value="accessKeySecret"
              @input="$emit('update:accessKeySecret', ($event.target as HTMLInputElement).value)"
              :type="showAccessKeySecret ? 'text' : 'password'"
              class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 pr-12 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
              placeholder="Access Key Secret"
          />
          <button
              type="button"
              @click="showAccessKeySecret = !showAccessKeySecret"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
          >
            <EyeIcon v-if="!showAccessKeySecret" class="h-5 w-5" />
            <EyeSlashIcon v-else class="h-5 w-5" />
          </button>
        </div>
      </div>
    </div>
    <div>
      <label class="block text-sm font-medium text-gray-300 mb-2">URL 前缀</label>
      <input
          :value="urlPrefix"
          @input="$emit('update:urlPrefix', ($event.target as HTMLInputElement).value)"
          type="text"
          class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
          placeholder="https://your-bucket.oss-cn-hangzhou.aliyuncs.com"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'

defineProps<{
  endpoint?: string
  bucket?: string
  accessKeyId?: string
  accessKeySecret?: string
  urlPrefix?: string
}>()

defineEmits<{
  (e: 'update:endpoint', value: string): void
  (e: 'update:bucket', value: string): void
  (e: 'update:accessKeyId', value: string): void
  (e: 'update:accessKeySecret', value: string): void
  (e: 'update:urlPrefix', value: string): void
}>()

const showAccessKeyId = ref(false)
const showAccessKeySecret = ref(false)
</script>
