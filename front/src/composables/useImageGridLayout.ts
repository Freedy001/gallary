import {computed, onMounted, onUnmounted, ref, type Ref, watch} from 'vue'
import {useUIStore} from '@/stores/ui'
import type {Image} from '@/types'

export function useImageGridLayout(images: Ref<(Image | null)[]>) {
  const uiStore = useUIStore()

  const currentColumnCount = ref(4)

  const isWaterfall = computed(() => uiStore.gridDensity >= 14)

  const waterfallImages = computed(() => {
    if (!isWaterfall.value) return []

    const cols: { image: Image | null, index: number }[][] =
      Array.from({ length: currentColumnCount.value }, () => [])

    images.value.forEach((image, index) => {
      const colIndex = index % currentColumnCount.value
      if (cols[colIndex]) cols[colIndex].push({ image, index })
    })

    return cols
  })

  const gridClass = computed(() => {
    const columns = uiStore.gridColumns
    const desktopClass = {
      1: 'md:grid-cols-1',
      2: 'md:grid-cols-2',
      3: 'md:grid-cols-3',
      4: 'md:grid-cols-4',
      5: 'md:grid-cols-5',
      6: 'md:grid-cols-6',
      7: 'md:grid-cols-7',
      8: 'md:grid-cols-8',
      9: 'md:grid-cols-9',
      10: 'md:grid-cols-10',
      11: 'md:grid-cols-11',
      12: 'md:grid-cols-12',
      13: 'md:grid-cols-13',
      14: 'md:grid-cols-14',
      15: 'md:grid-cols-15',
      16: 'md:grid-cols-16',
    }[columns.desktop] || 'md:grid-cols-4'

    const tabletClass = {
      1: 'sm:grid-cols-1',
      2: 'sm:grid-cols-2',
      3: 'sm:grid-cols-3',
      4: 'sm:grid-cols-4',
    }[columns.tablet] || 'sm:grid-cols-2'

    const mobileClass = columns.mobile === 1 ? 'grid-cols-1' : 'grid-cols-2'

    return `${mobileClass} ${tabletClass} ${desktopClass}`
  })

  function updateColumnCount() {
    const width = window.innerWidth
    const cols = uiStore.gridColumns
    if (width >= 768) {
      currentColumnCount.value = cols.desktop
    } else if (width >= 640) {
      currentColumnCount.value = cols.tablet
    } else {
      currentColumnCount.value = cols.mobile
    }
  }

  watch(() => uiStore.gridColumns, () => {
    updateColumnCount()
  })

  onMounted(() => {
    updateColumnCount()
    window.addEventListener('resize', updateColumnCount)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', updateColumnCount)
  })

  return {
    isWaterfall: isWaterfall,
    waterfallImages: waterfallImages,
    gridClass: gridClass,
    currentColumnCount: currentColumnCount
  }
}
