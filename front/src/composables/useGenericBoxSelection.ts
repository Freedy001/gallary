import type {Ref} from 'vue'
import {computed, onUnmounted, ref} from 'vue'

export interface GenericBoxSelectionOptions<T> {
  containerRef: Ref<HTMLElement | null | undefined>
  // Map of item index to element
  itemRefs: Map<number, HTMLElement>
  // Get the list of items
  getItems: () => T[]
  // Get item ID
  getItemId: (item: T) => number
  // Toggle selection for an ID
  toggleSelection: (id: number) => void
  onSelectionStart?: () => void
  onSelectionEnd?: () => void
  // Whether to use scroll position in calculations (for scrollable containers)
  useScroll?: boolean
}

export function useGenericBoxSelection<T>(options: GenericBoxSelectionOptions<T>) {
  const isSelecting = ref(false)
  const selectionStart = ref({ x: 0, y: 0 })
  const selectionCurrent = ref({ x: 0, y: 0 })
  let isDragOperation = false
  let isDraggingElement = false  // 标记是否正在拖拽可拖拽元素

  const selectionBoxStyle = computed(() => {
    if (!isSelecting.value) return null

    const left = Math.min(selectionStart.value.x, selectionCurrent.value.x)
    const top = Math.min(selectionStart.value.y, selectionCurrent.value.y)
    const width = Math.abs(selectionCurrent.value.x - selectionStart.value.x)
    const height = Math.abs(selectionCurrent.value.y - selectionStart.value.y)

    return {
      left: `${left}px`,
      top: `${top}px`,
      width: `${width}px`,
      height: `${height}px`
    }
  })

  function getPointInContainer(e: MouseEvent) {
    if (!options.containerRef.value) return { x: 0, y: 0 }

    const containerRect = options.containerRef.value.getBoundingClientRect()

    let x = e.clientX - containerRect.left
    let y = e.clientY - containerRect.top

    if (options.useScroll) {
      x += options.containerRef.value.scrollLeft
      y += options.containerRef.value.scrollTop
    }

    return { x, y }
  }

  function handleMouseDown(e: MouseEvent) {
    if (e.button !== 0 || !options.containerRef.value) return

    // 如果点击的是可拖拽元素，则不启动框选
    const target = e.target as HTMLElement
    if (target.closest('[draggable="true"]')) {
      isDraggingElement = true
      return
    }

    isDragOperation = false
    isDraggingElement = false

    const { x, y } = getPointInContainer(e)
    selectionStart.value = { x, y }
    selectionCurrent.value = { x, y }

    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
  }

  function handleMouseMove(e: MouseEvent) {
    // 如果正在拖拽元素，不处理框选
    if (isDraggingElement) return

    if (!isSelecting.value) {
      const { x, y } = getPointInContainer(e)
      const dx = x - selectionStart.value.x
      const dy = y - selectionStart.value.y

      // Threshold 5px squared
      if (dx * dx + dy * dy > 25) {
        isSelecting.value = true
        options.onSelectionStart?.()
      } else {
        return
      }
    }

    const { x, y } = getPointInContainer(e)
    selectionCurrent.value = { x, y }
  }

  function handleMouseUp() {
    window.removeEventListener('mousemove', handleMouseMove)
    window.removeEventListener('mouseup', handleMouseUp)

    // 重置拖拽元素标记
    isDraggingElement = false

    if (isSelecting.value) {
      isSelecting.value = false
      isDragOperation = true
      setTimeout(() => {
        isDragOperation = false
      }, 0)
      options.onSelectionEnd?.()
      updateSelection()
    }
  }

  function updateSelection() {
    if (!options.containerRef.value) return

    const containerRect = options.containerRef.value.getBoundingClientRect()

    // 框选区域（相对于容器）
    const left = Math.min(selectionStart.value.x, selectionCurrent.value.x)
    const top = Math.min(selectionStart.value.y, selectionCurrent.value.y)
    const right = Math.max(selectionStart.value.x, selectionCurrent.value.x)
    const bottom = Math.max(selectionStart.value.y, selectionCurrent.value.y)

    const items = options.getItems()

    options.itemRefs.forEach((el, index) => {
      const item = items[index]
      if (!item) return

      // 检查元素是否仍然在 DOM 中且可见
      if (!el.isConnected) return

      // 使用 getBoundingClientRect 获取元素在视口中的实际位置
      // 这在虚拟滚动场景下更可靠，因为 offsetLeft/offsetTop 依赖于 offsetParent
      const elRect = el.getBoundingClientRect()

      // 跳过不可见的元素（虚拟滚动中被回收的元素可能有无效的尺寸）
      if (elRect.width === 0 || elRect.height === 0) return

      // 检查元素是否在容器的可视范围内（虚拟滚动场景下的额外验证）
      // 元素的视口位置应该与容器有交集
      if (elRect.bottom < containerRect.top || elRect.top > containerRect.bottom ||
          elRect.right < containerRect.left || elRect.left > containerRect.right) {
        return
      }

      // 将元素位置转换为相对于容器的坐标
      let elLeft = elRect.left - containerRect.left
      let elTop = elRect.top - containerRect.top

      // 如果使用滚动，需要加上滚动偏移
      if (options.useScroll) {
        elLeft += options.containerRef.value!.scrollLeft
        elTop += options.containerRef.value!.scrollTop
      }

      const rect = {
        left: elLeft,
        top: elTop,
        right: elLeft + elRect.width,
        bottom: elTop + elRect.height
      }

      const isIntersecting = !(
        rect.right < left ||
        rect.left > right ||
        rect.bottom < top ||
        rect.top > bottom
      )
      if (!isIntersecting) return

      const id = options.getItemId(item)
      options.toggleSelection(id)
    })
  }

  // Clean up in case component unmounts while dragging
  onUnmounted(() => {
    window.removeEventListener('mousemove', handleMouseMove)
    window.removeEventListener('mouseup', handleMouseUp)
  })

  return {
    selectionBoxStyle: selectionBoxStyle,
    handleMouseDown: handleMouseDown,
    isSelecting: isSelecting,
    isDragOperation: () => isDragOperation
  }
}
