<template>
  <div class="space-y-4 pt-4 border-t border-white/5">
    <div>
      <label class="block text-sm font-medium text-gray-300 mb-2">存储路径</label>
      <input
          :value="basePath"
          @input="$emit('update:basePath', ($event.target as HTMLInputElement).value)"
          type="text"
          class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
          placeholder="例如: ./storage/images"
      />
      <p class="mt-2 text-xs text-gray-500">
        修改存储路径后保存配置，系统将自动迁移现有文件
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  basePath?: string
}>()

defineEmits<{
  (e: 'update:basePath', value: string): void
}>()
</script>
