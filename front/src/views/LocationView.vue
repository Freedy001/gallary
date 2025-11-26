<template>
  <AppLayout>
    <div class="h-full w-full relative">
      <!-- Map Container -->
      <div id="location-map-container" class="w-full h-full">
        <div v-if="!amapConfigured"
             class="absolute inset-0 flex items-center justify-center bg-gray-100 text-gray-500 text-sm p-4 text-center z-10">
          请在 .env 文件中配置 VITE_AMAP_KEY 和 VITE_AMAP_SECURITY_KEY
        </div>
        <div v-if="loading" class="absolute inset-0 flex items-center justify-center bg-white/80 z-20">
          <div class="text-center">
            <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto"></div>
            <p class="mt-4 text-sm text-gray-600">加载地图中...</p>
          </div>
        </div>
      </div>

      <!-- 图片查看器 -->
      <ImageViewer/>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import {computed, onMounted, onUnmounted, ref} from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import ImageViewer from '@/components/gallery/ImageViewer.vue'
import {imageApi} from '@/api/image'
import {useUIStore} from '@/stores/ui'
import type {ClusterResult, GeoBounds} from '@/types'
import AMapLoader from '@amap/amap-jsapi-loader'
import "@amap/amap-jsapi-types";

import {useImageStore} from "@/stores/image.ts";

let map: any = null
let markers: any[] = []
const loading = ref(true)
const amapConfigured = computed(() => !!import.meta.env.VITE_AMAP_KEY)

const uiStore = useUIStore()
const imageStore = useImageStore()

// 更新聚合点
const updateClusters = async () => {
  if (!map) return

  try {
    const bounds = map.getBounds()
    const zoom = Math.floor(map.getZoom())

    const minLat = bounds.getSouthWest().lat
    const maxLat = bounds.getNorthEast().lat
    const minLng = bounds.getSouthWest().lng
    const maxLng = bounds.getNorthEast().lng

    const response = await imageApi.getClusters(minLat, maxLat, minLng, maxLng, zoom)
    renderMarkers(response.data || [])
  } catch (error) {
    console.error('获取聚合数据失败:', error)
  }
}

// 渲染标记
const renderMarkers = (clusters: ClusterResult[]) => {
  if (!map) return

  // 清除现有标记
  map.remove(markers)
  markers = []

  const newMarkers: any[] = []

  clusters.forEach(cluster => {
    const {latitude, longitude, count, cover_image, min_lat, max_lat, min_lng, max_lng} = cluster
    if (!cover_image) return

    const imageUrl = imageApi.getImageUrl(cover_image.thumbnail_path || cover_image.storage_path)

    // 创建自定义内容
    const content = `
      <div style="position: relative; width: 48px; height: 48px; cursor: pointer;">
        <div style="width: 100%; height: 100%; border-radius: 8px; border: 2px solid white; box-shadow: 0 4px 6px rgba(0,0,0,0.1); overflow: hidden; background: #f3f4f6;">
           <img src="${imageUrl}" style="width: 100%; height: 100%; object-fit: cover;" alt="-">
        </div>
        ${count > 1 ? `<div style="position: absolute; top: -8px; right: -8px; background: #3b82f6; color: white; border-radius: 9999px; padding: 0 6px; font-size: 12px; font-weight: bold; border: 2px solid white; min-width: 20px; text-align: center;">${count}</div>` : ''}
      </div>
    `

    const marker = new (window as any).AMap.Marker({
      position: [longitude, latitude],
      content: content,
      offset: new (window as any).AMap.Pixel(-24, -24), // Center the marker (48/2 = 24)
      zIndex: 100,
      extData: cluster,
      cursor: 'pointer'
    })

    // 添加点击事件
    marker.on('click', async () => {
      try {
        // 显示加载状态
        uiStore.setGlobalLoading(true, '加载图片中...')

        await imageStore.refreshImages(async (page, size) => (await imageApi.getClusterImages(min_lat, max_lat, min_lng, max_lng, page, size)).data, uiStore.pageSize)
        imageStore.viewerIndex = 1
      } catch (error) {
        console.error('加载聚合图片失败:', error)
      } finally {
        uiStore.setGlobalLoading(false)
      }
    })

    newMarkers.push(marker)
  })

  map.add(newMarkers)
  markers = newMarkers
}


const initMap = async () => {
  if (!amapConfigured.value) {
    loading.value = false
    return
  }

  try {
    (window as any)._AMapSecurityConfig = {
      securityJsCode: import.meta.env.VITE_AMAP_SECURITY_KEY,
    }

    const AMap = await AMapLoader.load({
      key: import.meta.env.VITE_AMAP_KEY,
      version: "2.0",
      plugins: [],
    })

    // 获取图片地理边界来决定初始中心和缩放
    let centerLng = 104.195397  // 默认中国中心
    let centerLat = 35.86166
    let zoom = 4
    let geoBounds: GeoBounds | null = null

    try {
      const boundsRes = await imageApi.getGeoBounds()
      if (boundsRes.data) {
        geoBounds = boundsRes.data
        // 计算中心点作为初始值
        centerLat = (geoBounds.min_lat + geoBounds.max_lat) / 2
        centerLng = (geoBounds.min_lng + geoBounds.max_lng) / 2
      }
    } catch (e) {
      console.warn('获取地理边界失败，使用默认位置', e)
    }

    map = new AMap.Map("location-map-container", {
      viewMode: "3D",
      layers: [
        // 卫星
        new AMap.TileLayer.Satellite(),
        // 路网
        new AMap.TileLayer.RoadNet()
      ],
      zoom: zoom,
      center: [centerLng, centerLat],
      resizeEnable: true,
    })

    map.on('complete', () => {
      loading.value = false
      // 如果有地理边界，使用 setBounds 自动适配
      if (geoBounds) {
        const bounds = new (window as any).AMap.Bounds(
          [geoBounds.min_lng, geoBounds.min_lat],  // 西南角
          [geoBounds.max_lng, geoBounds.max_lat]   // 东北角
        )
        map.setBounds(bounds, false, [100, 100, 100, 100])  // 添加 60px 边距
      }
      updateClusters()
    })

    map.on('moveend', updateClusters)
    map.on('zoomend', updateClusters)

  } catch (error) {
    console.error('地图初始化失败:', error)
    loading.value = false
  }
}

const destroyMap = () => {
  if (map) {
    map.destroy()
    map = null
    markers = []
  }
}

onMounted(async () => {
  await initMap()
})

onUnmounted(() => {
  destroyMap()
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>