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

    isDragOperation = false

    const { x, y } = getPointInContainer(e)
    selectionStart.value = { x, y }
    selectionCurrent.value = { x, y }

    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
  }

  function handleMouseMove(e: MouseEvent) {
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
    const left = Math.min(selectionStart.value.x, selectionCurrent.value.x)
    const top = Math.min(selectionStart.value.y, selectionCurrent.value.y)
    const right = Math.max(selectionStart.value.x, selectionCurrent.value.x)
    const bottom = Math.max(selectionStart.value.y, selectionCurrent.value.y)

    const items = options.getItems()

    options.itemRefs.forEach((el, index) => {
      const item = items[index]
      if (!item) return

      const rect = {
        left: el.offsetLeft,
        top: el.offsetTop,
        right: el.offsetLeft + el.offsetWidth,
        bottom: el.offsetTop + el.offsetHeight
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
