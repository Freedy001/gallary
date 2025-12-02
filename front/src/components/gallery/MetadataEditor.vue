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
        <label class="block text-sm font-medium text-black ">文件名</label>
        <input
          v-model="form.original_name"
          type="text"
          class="input"
          placeholder="输入文件名称"
        />
      </div>

      <!-- Location and Map -->
      <div class="space-y-4">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div class="relative">
              <label class="block text-sm font-medium text-black">地点名称 (搜索)</label>
              <input
                id="location-input"
                v-model="form.location_name"
                @input="onLocationInput"
                type="text"
                class="input"
                placeholder="输入关键字搜索或填写地点"
                autocomplete="off"
              />
              <!-- Custom Suggestions List -->
              <ul v-if="showSuggestions && suggestions.length > 0" class="absolute z-50 w-full mt-1 bg-white/95 backdrop-blur-sm border border-gray-200 rounded-lg shadow-xl max-h-60 overflow-y-auto">
                  <li
                      v-for="(item, index) in suggestions"
                      :key="index"
                      @click="selectSuggestion(item)"
                      class="px-3 py-2 text-sm text-gray-800 hover:bg-gray-100 cursor-pointer transition-colors duration-150"
                  >
                      <div class="font-medium">{{ item.name }}</div>
                      <div class="text-xs text-gray-500 truncate" v-if="item.district || item.address">
                          {{ item.district }}{{ item.address && typeof item.address === 'string' ? ' - ' + item.address : '' }}
                      </div>
                  </li>
              </ul>
            </div>
            <div>
              <label class="block text-sm font-medium text-black">经纬度</label>
              <div class="flex space-x-2">
                <input
                  v-model.number="form.latitude"
                  type="number"
                  step="any"
                  class="input bg-gray-50 cursor-not-allowed"
                  placeholder="纬度"
                  readonly
                />
                <input
                  v-model.number="form.longitude"
                  type="number"
                  step="any"
                  class="input bg-gray-50 cursor-not-allowed"
                  placeholder="经度"
                  readonly
                />
              </div>
            </div>
        </div>

        <!-- Map Container -->
        <div id="amap-container" class="w-full h-64 rounded-md border border-gray-300 relative">
            <div v-if="!amapConfigured" class="absolute inset-0 flex items-center justify-center bg-gray-100 bg-opacity-80 text-gray-500 text-sm p-4 text-center z-10">
                请在 .env 文件中配置 VITE_AMAP_KEY 和 VITE_AMAP_SECURITY_KEY
            </div>
        </div>
      </div>

      <div>
        <label class="block text-sm font-medium text-black">标签 (逗号分隔)</label>
        <input
          v-model="tagsInput"
          type="text"
          class="input"
          placeholder="风景, 2023"
        />
      </div>

      <!-- Metadata Key-Value Pairs -->
      <div>
        <div class="flex justify-between items-center mb-2">
          <label class="block text-sm font-medium text-black">扩展元数据</label>
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
              class="input"
              placeholder="键名"
            />
            <input
              v-model="item.value"
              type="text"
              class="input"
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
import { ref, computed, watch, shallowRef, nextTick, onUnmounted } from 'vue'
import Modal from '@/components/common/Modal.vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import type { Image, UpdateMetadataRequest, MetadataUpdate } from '@/types'
import { imageApi } from '@/api/image'
import AMapLoader from '@amap/amap-jsapi-loader'
import { useDebounceFn } from '@vueuse/core'
import { onClickOutside } from '@vueuse/core'

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
const map = shallowRef<any>(null)
const marker = shallowRef<any>(null)
const autoComplete = shallowRef<any>(null)
const amapConfigured = computed(() => !!import.meta.env.VITE_AMAP_KEY)

// Custom suggestions state
const suggestions = ref<any[]>([])
const showSuggestions = ref(false)
const locationInputRef = ref(null)

// Click outside to close suggestions
onClickOutside(locationInputRef, () => {
    showSuggestions.value = false
})

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
watch(() => props.modelValue, async (val) => {
    if (val) {
        if (isSingleMode.value && props.initialData) {
            // Pre-fill for single image
            form.value = {
                original_name: props.initialData.original_name,
                location_name: props.initialData.location_name || undefined,
                latitude: props.initialData.latitude || undefined,
                longitude: props.initialData.longitude || undefined,
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

        suggestions.value = []
        showSuggestions.value = false

        await nextTick()
        if (amapConfigured.value) {
            initMap()
        }
    } else {
        destroyMap()
    }
})

const initMap = () => {
    (window as any)._AMapSecurityConfig = {
        securityJsCode: import.meta.env.VITE_AMAP_SECURITY_KEY,
    };

    AMapLoader.load({
        key: import.meta.env.VITE_AMAP_KEY,
        version: "2.0",
        plugins: ['AMap.AutoComplete', 'AMap.PlaceSearch', 'AMap.Geocoder'],
    }).then((AMap) => {
        // Init map
        const hasLocation = form.value.longitude && form.value.latitude
        const center = hasLocation
            ? [form.value.longitude, form.value.latitude]
            : [116.397428, 39.90923]; // Default to Beijing

        map.value = new AMap.Map("amap-container", {
            viewMode: "3D",
            zoom: hasLocation ? 16 : 11,
            center: center,
        });

        // Init marker if exists
        if (hasLocation) {
            addMarker(new AMap.LngLat(form.value.longitude, form.value.latitude), AMap)
        }

        // Click to pick location
        map.value.on('click', (e: any) => {
             updateLocationFromLngLat(e.lnglat, AMap);
             showSuggestions.value = false;
        });

        // AutoComplete - NOT linked to input anymore
        autoComplete.value = new AMap.AutoComplete({
             // city: '全国' // default
        });

    }).catch(e => {
        console.error('AMap load failed', e);
    })
}

// Debounced search input handler
const onLocationInput = useDebounceFn(() => {
    if (!form.value.location_name || !autoComplete.value) {
        suggestions.value = []
        showSuggestions.value = false
        return
    }

    autoComplete.value.search(form.value.location_name, (status: string, result: any) => {
        if (status === 'complete' && result.tips) {
            suggestions.value = result.tips;
            showSuggestions.value = true;
        } else {
            suggestions.value = [];
            showSuggestions.value = false;
        }
    })
}, 300)

const selectSuggestion = (item: any) => {
    form.value.location_name = item.name;
    showSuggestions.value = false;

    if (item.location && item.location.lng && item.location.lat) {
         // We have location directly
         updateFormLocation(item.location.lng, item.location.lat);

         // Update map
         if (map.value) {
             map.value.setZoomAndCenter(16, [item.location.lng, item.location.lat]);

             // Ensure AMap is loaded before adding marker (it should be if map exists)
             // We need the AMap constructor which is not globally available easily here unless stored
             // But we can use the map instance's constructor or just assume window.AMap if loaded via loader
             // Or cleaner: store AMap constructor in a ref or just use map context
             // For now, assuming window.AMap is available after load
             if ((window as any).AMap) {
                addMarker(item.location, (window as any).AMap);
             }
         }
    } else if (item.adcode && map.value) {
         // Only city/district info
         map.value.setCity(item.adcode);
    }
}

const updateLocationFromLngLat = (lnglat: any, AMap: any) => {
    updateFormLocation(lnglat.lng, lnglat.lat);
    addMarker(lnglat, AMap);

    // Reverse geocoding to get name
    const geocoder = new AMap.Geocoder();
    geocoder.getAddress(lnglat, (status: string, result: any) => {
        if (status === 'complete' && result.regeocode) {
             form.value.location_name = result.regeocode.formattedAddress;
        }
    });
}

const addMarker = (lnglat: any, AMap: any) => {
    if (marker.value) {
        marker.value.setPosition(lnglat);
    } else {
        marker.value = new AMap.Marker({
            position: lnglat,
            map: map.value
        });
    }
}

const updateFormLocation = (lng: number, lat: number) => {
    form.value.longitude = lng;
    form.value.latitude = lat;
}

const destroyMap = () => {
    if (map.value) {
        map.value.destroy();
        map.value = null;
        marker.value = null;
        autoComplete.value = null;
    }
}

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

onUnmounted(() => {
    destroyMap()
})
</script>

<style>
.input {
  margin-top: 0.25rem;
  display: block;
  width: 100%;
  border-radius: 0.375rem;
  box-shadow: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  padding: 0.5rem 0.75rem;
  color: black;
}

.input::-webkit-input-placeholder {
  color: rgba(81, 81, 81, 0.82);
}

.input:focus {
  border: rgba(255, 255, 255, 0.6);
  outline: 2px solid transparent;
  outline-offset: 2px;
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.6);
}

@media (min-width: 640px) {
  .input {
    font-size: 0.875rem;
    line-height: 1.25rem;
  }
}

@media (prefers-color-scheme: dark) {
  .input {
    background: linear-gradient(135deg, rgba(255, 255, 255, 0.5) 0%, rgba(255, 255, 255, 0.2) 100%);
    border-color: rgba(112, 112, 112, 0.2);
  }
}
</style>