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
            placeholder="localhost:9000"
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
        <label class="block text-sm font-medium text-gray-300 mb-2">Secret Access Key</label>
        <div class="relative">
          <input
              :value="secretAccessKey"
              @input="$emit('update:secretAccessKey', ($event.target as HTMLInputElement).value)"
              :type="showSecretAccessKey ? 'text' : 'password'"
              class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 pr-12 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
              placeholder="Secret Access Key"
          />
          <button
              type="button"
              @click="showSecretAccessKey = !showSecretAccessKey"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
          >
            <EyeIcon v-if="!showSecretAccessKey" class="h-5 w-5" />
            <EyeSlashIcon v-else class="h-5 w-5" />
          </button>
        </div>
      </div>
    </div>
    <div class="flex items-center gap-3">
      <label class="relative inline-flex items-center cursor-pointer">
        <input
            :checked="useSsl"
            @change="$emit('update:useSsl', ($event.target as HTMLInputElement).checked)"
            type="checkbox"
            class="sr-only peer"
        />
        <div
            class="w-11 h-6 bg-white/10 rounded-full peer peer-checked:bg-primary-500/50 peer-focus:ring-2 peer-focus:ring-primary-500/30 transition-colors after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full"
        ></div>
      </label>
      <span class="text-sm text-gray-300">使用 SSL</span>
    </div>
    <div>
      <label class="block text-sm font-medium text-gray-300 mb-2">URL 前缀</label>
      <input
          :value="urlPrefix"
          @input="$emit('update:urlPrefix', ($event.target as HTMLInputElement).value)"
          type="text"
          class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
          placeholder="http://localhost:9000/bucket"
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
  secretAccessKey?: string
  useSsl?: boolean
  urlPrefix?: string
}>()

defineEmits<{
  (e: 'update:endpoint', value: string): void
  (e: 'update:bucket', value: string): void
  (e: 'update:accessKeyId', value: string): void
  (e: 'update:secretAccessKey', value: string): void
  (e: 'update:useSsl', value: boolean): void
  (e: 'update:urlPrefix', value: string): void
}>()

const showAccessKeyId = ref(false)
const showSecretAccessKey = ref(false)
</script>
