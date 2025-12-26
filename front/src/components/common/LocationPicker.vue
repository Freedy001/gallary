<template>
  <div class="space-y-4">
    <!-- 搜索输入框 -->
    <div class="relative" ref="locationInputRef">
      <label v-if="label" class="block text-sm font-medium text-white/80 mb-2">{{ label }}</label>
      <input
        v-model="locationName"
        @input="onLocationInput"
        type="text"
        :placeholder="placeholder"
        class="w-full rounded-xl border border-white/10 bg-white/5 px-4 py-2.5 text-sm text-white transition-colors placeholder:text-gray-600 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
        autocomplete="off"
      />
      <!-- 搜索建议列表 -->
      <ul v-if="showSuggestions && suggestions.length > 0"
          class="absolute z-50 w-full mt-1 bg-[#18181b]/95 backdrop-blur-xl border border-white/10 rounded-xl shadow-2xl max-h-60 overflow-y-auto ring-1 ring-black/50">
        <li
          v-for="(item, index) in suggestions"
          :key="index"
          @click="selectSuggestion(item)"
          class="px-4 py-3 text-sm text-gray-200 hover:text-white hover:bg-white/10 cursor-pointer transition-colors duration-150 border-b border-white/5 last:border-0"
        >
          <div class="font-medium">{{ item.name }}</div>
          <div class="text-xs text-white/50 truncate" v-if="item.district || item.address">
            {{ item.district }}{{ item.address && typeof item.address === 'string' ? ' - ' + item.address : '' }}
          </div>
        </li>
      </ul>
    </div>

    <!-- 地图容器 -->
    <div v-if="showMap" id="amap-container" class="w-full h-64 rounded-xl border border-white/10 relative overflow-hidden">
      <div v-if="!amapConfigured" class="absolute inset-0 flex items-center justify-center bg-black/40 backdrop-blur-sm text-white/70 text-sm p-4 text-center z-10">
        请在 .env 文件中配置 VITE_AMAP_KEY 和 VITE_AMAP_SECURITY_KEY
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, shallowRef, watch, onUnmounted, onMounted } from 'vue'
import AMapLoader from '@amap/amap-jsapi-loader'
import { useDebounceFn, onClickOutside } from '@vueuse/core'

const props = withDefaults(defineProps<{
  modelValue?: string // location name
  latitude?: number
  longitude?: number
  label?: string
  placeholder?: string
  showMap?: boolean
}>(), {
  placeholder: '输入关键字搜索或在地图上选择',
  showMap: true
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'update:latitude', value: number): void
  (e: 'update:longitude', value: number): void
  (e: 'select', item: any): void
}>()

// State
const locationName = ref(props.modelValue || '')
const suggestions = ref<any[]>([])
const showSuggestions = ref(false)
const locationInputRef = ref(null)

// Map related
let map: AMap.Map | null = null
let marker: AMap.Marker | null = null
let autoComplete = shallowRef<any>(null)
const amapConfigured = computed(() => !!import.meta.env.VITE_AMAP_KEY)

// Sync internal state with props
watch(() => props.modelValue, (val) => {
  if (val !== locationName.value) {
    locationName.value = val || ''
  }
})

// Update location on map if props change
watch([() => props.latitude, () => props.longitude], ([lat, lng]) => {
  if (lat && lng && map && (!marker || !isSameLocation(marker.getPosition(), lng, lat))) {
    updateMapLocation(lng, lat)
  }
})

// Click outside to close suggestions
onClickOutside(locationInputRef, () => {
  showSuggestions.value = false
})

onMounted(() => {
  if (amapConfigured.value && props.showMap) {
    initMap()
  } else if (amapConfigured.value && !props.showMap) {
    // Only init autocomplete if map is hidden
    initAutoComplete()
  }
})

onUnmounted(() => {
  destroyMap()
})

const initAutoComplete = () => {
  (window as any)._AMapSecurityConfig = {
    securityJsCode: import.meta.env.VITE_AMAP_SECURITY_KEY,
  };

  AMapLoader.load({
    key: import.meta.env.VITE_AMAP_KEY,
    version: "2.0",
    plugins: ['AMap.AutoComplete'],
  }).then((AMap) => {
    autoComplete.value = new AMap.AutoComplete({});
  }).catch(e => {
    console.error('AMap load failed', e);
  })
}

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
    const hasLocation = props.longitude && props.latitude
    const center = hasLocation
        ? [props.longitude, props.latitude]
        : [116.397428, 39.90923]; // Default to Beijing

    map = new AMap.Map("amap-container", {
      viewMode: "3D",
      zoom: hasLocation ? 16 : 11,
      center: center,
      mapStyle: "amap://styles/dark", // Obsidian theme compatible
    });

    // Init marker if exists
    if (hasLocation) {
      addMarker(new AMap.LngLat(props.longitude, props.latitude), AMap)
    }

    // Click to pick location
    (map as AMap.Map).on('click', (e: any) => {
      updateLocationFromLngLat(e.lnglat, AMap);
      showSuggestions.value = false;
    });

    // AutoComplete
    autoComplete.value = new AMap.AutoComplete({});

  }).catch(e => {
    console.error('AMap load failed', e);
  })
}

// Helper to check if location is roughly the same (avoid loops)
const isSameLocation = (lnglat: any, lng: number, lat: number) => {
  if (!lnglat) return false
  return Math.abs(lnglat.lng - lng) < 0.00001 && Math.abs(lnglat.lat - lat) < 0.00001
}

const updateMapLocation = (lng: number, lat: number) => {
  if (!map) return

  // @ts-ignore
  const AMap = window.AMap
  if (!AMap) return

  const lnglat = new AMap.LngLat(lng, lat)
  map.setZoomAndCenter(16, lnglat)
  addMarker(lnglat, AMap)
}

const addMarker = (lnglat: any, AMap: any) => {
  if (marker) {
    marker.setPosition(lnglat);
  } else {
    marker = new AMap.Marker({
      position: lnglat,
      map: map
    });
  }
}

const updateLocationFromLngLat = (lnglat: any, AMap: any) => {
  emit('update:longitude', lnglat.lng)
  emit('update:latitude', lnglat.lat)
  addMarker(lnglat, AMap);

  // Reverse geocoding to get name
  const geocoder = new AMap.Geocoder();
  geocoder.getAddress(lnglat, (status: string, result: any) => {
    if (status === 'complete' && result.regeocode) {
      locationName.value = result.regeocode.formattedAddress;
      emit('update:modelValue', locationName.value)
    }
  });
}

// Debounced search input handler
const onLocationInput = useDebounceFn(() => {
  emit('update:modelValue', locationName.value)

  if (!locationName.value || !autoComplete.value) {
    suggestions.value = []
    showSuggestions.value = false
    return
  }

  autoComplete.value.search(locationName.value, (status: string, result: any) => {
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
  locationName.value = item.name;
  emit('update:modelValue', item.name)
  emit('select', item)

  showSuggestions.value = false;

  if (item.location && item.location.lng && item.location.lat) {
    emit('update:longitude', item.location.lng)
    emit('update:latitude', item.location.lat)

    // Update map if visible
    if (map) {
      // @ts-ignore
      const AMap = window.AMap
      if (AMap) {
        map.setZoomAndCenter(16, [item.location.lng, item.location.lat]);
        addMarker(item.location, AMap);
      }
    }
  }
}

const destroyMap = () => {
  if (map) {
    map.destroy();
    map = null;
    marker = null;
    autoComplete.value = null;
  }
}
</script>

<style>
#amap-container {
  background-image: none !important;
  background-color: rgb(42, 42, 42) !important; /* 确保背景色和你想要的深色一致 */
}
</style>