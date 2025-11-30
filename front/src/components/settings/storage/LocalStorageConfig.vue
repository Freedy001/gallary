<template>
  <div class="space-y-4 pt-4 border-t border-white/5">
    <!-- 迁移进度 -->
    <MigrationProgress ref="migrationProgressRef" />

    <div>
      <label class="block text-sm font-medium text-gray-300 mb-2">存储路径</label>
      <input
          :value="basePath"
          @input="$emit('update:basePath', ($event.target as HTMLInputElement).value)"
          type="text"
          class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
          placeholder="例如: ./storage/images"
      />
    </div>
    <div>
      <label class="block text-sm font-medium text-gray-300 mb-2">URL 前缀</label>
      <input
          :value="urlPrefix"
          @input="$emit('update:urlPrefix', ($event.target as HTMLInputElement).value)"
          type="text"
          class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
          placeholder="例如: /static/images"
      />
    </div>

    <!-- 迁移按钮 -->
    <div class="pt-2">
      <button
          @click="startMigration"
          :disabled="migrating || !canMigrate"
          class="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 ring-1 ring-blue-500/30 transition-all duration-300 text-sm disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': migrating }" />
        {{ migrating ? '启动中...' : '迁移现有文件到新路径' }}
      </button>
      <p class="mt-2 text-xs text-gray-500">
        修改存储路径后，点击此按钮可将现有文件迁移到新位置
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import { migrationApi } from '@/api/migration'
import { useDialogStore } from '@/stores/dialog'
import MigrationProgress from '../MigrationProgress.vue'

const props = defineProps<{
  basePath?: string
  urlPrefix?: string
  originalBasePath: string
  originalUrlPrefix: string
}>()

const emit = defineEmits<{
  (e: 'update:basePath', value: string): void
  (e: 'update:urlPrefix', value: string): void
  (e: 'migration-started'): void
}>()

const dialogStore = useDialogStore()
const migrating = ref(false)
const migrationProgressRef = ref<InstanceType<typeof MigrationProgress> | null>(null)

const canMigrate = computed(() => {
  return props.basePath &&
      props.urlPrefix &&
      (props.basePath !== props.originalBasePath ||
          props.urlPrefix !== props.originalUrlPrefix)
})

async function startMigration() {
  if (!props.basePath || !props.urlPrefix) {
    await dialogStore.alert({ title: '错误', message: '请填写存储路径和URL前缀', type: 'error' })
    return
  }

  const confirmed = await dialogStore.confirm({
    title: '启动存储迁移',
    message: `确定要将文件从 "${props.originalBasePath}" 迁移到 "${props.basePath}" 吗？\n\n迁移期间图片将暂时无法访问。`,
    type: 'warning'
  })

  if (!confirmed) return

  migrating.value = true
  try {
    await migrationApi.start({
      new_base_path: props.basePath,
      new_url_prefix: props.urlPrefix,
    })

    if (migrationProgressRef.value) {
      migrationProgressRef.value.refresh()
    }

    await dialogStore.alert({
      title: '已启动',
      message: '存储迁移任务已启动，请查看下方进度',
      type: 'success'
    })

    emit('migration-started')
  } catch (error: any) {
    await dialogStore.alert({
      title: '错误',
      message: error.message || '启动迁移失败',
      type: 'error'
    })
  } finally {
    migrating.value = false
  }
}
</script>
