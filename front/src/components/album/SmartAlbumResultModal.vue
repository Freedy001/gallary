<template>
  <Modal v-model="isOpen" size="lg" title="智能相册生成结果">
    <!-- 生成结果 -->
    <div v-if="smartAlbumStore.result" class="space-y-4">
      <div class="p-4 bg-green-500/10 border border-green-500/20 rounded-xl">
        <div class="flex items-center gap-2 text-green-400 mb-2">
          <CheckCircleIcon class="h-5 w-5"/>
          <span class="font-medium">生成完成</span>
        </div>
        <div class="text-sm text-gray-300 space-y-1">
          <p>已创建 <span class="text-white font-semibold text-base tabular-nums">{{
              smartAlbumStore.result.cluster_count
            }}</span> 个智能相册</p>
          <p>共处理 <span class="text-white font-semibold text-base tabular-nums">{{
              smartAlbumStore.result.total_images
            }}</span> 张图片</p>
        </div>
      </div>

      <!-- 噪声图片展示 -->
      <div v-if="smartAlbumStore.result.noise_count && smartAlbumStore.result.noise_count > 0">
        <button
            class="w-full flex items-center justify-between px-4 py-3 text-sm text-gray-300 bg-white/5 hover:bg-white/10 rounded-xl transition-colors"
            @click="toggleNoiseImages"
        >
          <div class="flex items-center gap-2">
            <ExclamationTriangleIcon class="h-4 w-4 text-amber-400"/>
            <span>{{ smartAlbumStore.result.noise_count }} 张图片未被归类（噪声点）</span>
          </div>
          <ChevronDownIcon :class="['h-4 w-4 transition-transform', showNoiseImages ? 'rotate-180' : '']"/>
        </button>

        <Transition
            enter-active-class="transition-all duration-200 ease-out"
            enter-from-class="opacity-0 max-h-0"
            enter-to-class="opacity-100 max-h-[400px]"
            leave-active-class="transition-all duration-150 ease-in"
            leave-from-class="opacity-100 max-h-[400px]"
            leave-to-class="opacity-0 max-h-0"
        >
          <div v-show="showNoiseImages" class="mt-2 overflow-hidden">
            <div v-if="loadingNoiseImages" class="flex items-center justify-center py-8">
              <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-primary-500"></div>
            </div>
            <div v-else-if="noiseImages.length > 0"
                 class="grid grid-cols-6 gap-2 max-h-[300px] overflow-y-auto p-2 bg-white/5 rounded-xl">
              <div
                  v-for="image in noiseImages"
                  :key="image.id"
                  class="relative aspect-square rounded-lg overflow-hidden group cursor-pointer bg-gray-800"
              >
                <img
                    :alt="image.original_name"
                    :src="image.thumbnail_url || image.url"
                    class="w-full h-full object-cover transition-transform duration-200 group-hover:scale-105"
                    loading="lazy"
                />
                <div
                    class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                  <span class="text-xs text-white truncate px-1">{{ image.original_name }}</span>
                </div>
              </div>
            </div>
            <p v-else class="text-sm text-gray-500 text-center py-4">无法加载噪声图片</p>
          </div>
        </Transition>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="smartAlbumStore.errorMessage" class="p-4 bg-red-500/10 border border-red-500/20 rounded-xl">
      <div class="flex items-center gap-2 text-red-400 mb-2">
        <XCircleIcon class="h-5 w-5"/>
        <span class="font-medium">生成失败</span>
      </div>
      <p class="text-sm text-gray-300">{{ smartAlbumStore.errorMessage }}</p>
    </div>

    <!-- 操作按钮 -->
    <div class="flex justify-end gap-3 pt-4">
      <button
          class="px-5 py-2.5 rounded-xl bg-white/10 text-white hover:bg-white/20 transition-colors"
          type="button"
          @click="handleClose"
      >
        确定
      </button>
    </div>
  </Modal>
</template>

<script lang="ts" setup>
import {computed, ref, watch} from 'vue'
import {CheckCircleIcon, ChevronDownIcon, ExclamationTriangleIcon, XCircleIcon} from '@heroicons/vue/24/outline'
import Modal from '@/components/widgets/common/Modal.vue'
import {useSmartAlbumStore} from '@/stores/smartAlbum'
import {imageApi} from '@/api/image'
import type {Image} from '@/types'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const smartAlbumStore = useSmartAlbumStore()

const isOpen = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const showNoiseImages = ref(false)
const loadingNoiseImages = ref(false)
const noiseImages = ref<Image[]>([])

// 切换噪声图片展示
async function toggleNoiseImages() {
  showNoiseImages.value = !showNoiseImages.value

  // 如果展开且还没加载过，则加载噪声图片
  if (showNoiseImages.value && noiseImages.value.length === 0 && smartAlbumStore.result?.noise_image_ids?.length) {
    await loadNoiseImages()
  }
}

// 加载噪声图片
async function loadNoiseImages() {
  const ids = smartAlbumStore.result?.noise_image_ids
  if (!ids || ids.length === 0) return

  try {
    loadingNoiseImages.value = true
    const res = await imageApi.getByIds(ids)
    noiseImages.value = res.data || []
  } catch (err) {
    console.error('加载噪声图片失败', err)
  } finally {
    loadingNoiseImages.value = false
  }
}

// 监听弹窗打开/关闭，重置状态
watch(isOpen, (val) => {
  if (!val) {
    showNoiseImages.value = false
    noiseImages.value = []
  }
})

function handleClose() {
  isOpen.value = false
  // 关闭弹窗时，清除结果状态，这样 Sidebar 上的提示也会消失
  smartAlbumStore.resetState()
}
</script>
