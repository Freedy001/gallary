import {onActivated} from 'vue'
import {onBeforeRouteLeave} from 'vue-router'

/**
 * 在 keep-alive 缓存的组件中保存和恢复滚动位置
 * @param scrollContainerId 滚动容器的 ID，默认为 'main-scroll-container'
 */
export function useScrollPosition(scrollContainerId = 'main-scroll-container') {
  let savedScrollTop = 0

  // 在路由离开前保存滚动位置（此时 DOM 还在）
  onBeforeRouteLeave(() => {
    const container = document.getElementById(scrollContainerId)
    if (container) {
      savedScrollTop = container.scrollTop
    }
  })

  onActivated(() => {
    const container = document.getElementById(scrollContainerId)
    if (container && savedScrollTop > 0) {
      // 临时禁用平滑滚动，立即恢复位置
      const originalScrollBehavior = container.style.scrollBehavior
      container.style.scrollBehavior = 'auto'
      container.scrollTop = savedScrollTop
      // 恢复原始滚动行为
      container.style.scrollBehavior = originalScrollBehavior
    }
  })
}
