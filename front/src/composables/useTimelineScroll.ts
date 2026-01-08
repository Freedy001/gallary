import {onMounted, onUnmounted, ref, type Ref} from 'vue'
import {useDebounceFn, useThrottleFn} from '@vueuse/core'
import {useUIStore} from '@/stores/ui'
import type {Image} from '@/types'

export interface UseTimelineScrollOptions {
  images: Ref<(Image | null)[]>
  scrollContainerId?: string
}

export function useTimelineScroll(options: UseTimelineScrollOptions) {
  const {images, scrollContainerId = 'main-scroll-container'} = options
  const uiStore = useUIStore()

  const scrollContainer = ref<HTMLElement | null>(null)

  const updateActiveDate = useThrottleFn(() => {
    if (!scrollContainer.value) return

    const container = scrollContainer.value
    const rect = container.getBoundingClientRect()

    const checkX = rect.left + 100
    const checkY = rect.top + 100

    const el = document.elementFromPoint(checkX, checkY)
    if (!el) return

    const itemEl = el.closest('[data-index]') as HTMLElement
    if (itemEl && itemEl.dataset.index) {
      const image = images.value[parseInt(itemEl.dataset.index)] as (Image | null)
      if (image) {
        const date = image.taken_at || image.created_at
        if (date && date !== uiStore.timeLineState?.date) {
          uiStore.setTimeLineState({date, location: image.location_name})
        }
      }
    }
  }, 100)

  const hideTimeline = useDebounceFn(() => {
    uiStore.setTimeLineState(null)
  }, 1500)

  function handleScroll() {
    updateActiveDate()
    hideTimeline()
  }

  onMounted(() => {
    scrollContainer.value = document.getElementById(scrollContainerId)
    if (scrollContainer.value) {
      scrollContainer.value.addEventListener('scroll', handleScroll)
      setTimeout(() => handleScroll(), 100)
    }
  })

  onUnmounted(() => {
    if (scrollContainer.value) {
      scrollContainer.value.removeEventListener('scroll', handleScroll)
    }
  })

  return {
    scrollContainer: scrollContainer
  }
}
