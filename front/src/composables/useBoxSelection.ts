import {ref, computed, onUnmounted} from 'vue'
import type {Ref} from 'vue'
import {useImageStore} from "@/stores/image.ts";

const imageStore = useImageStore()

export interface BoxSelectionOptions {
  containerRef: Ref<HTMLElement | null | undefined>
  // Callback to get item rects, called on mouse down
  itemRefs: Map<number, HTMLElement>
  onSelectionStart?: () => void
  // Callback when selection ends (optional)
  onSelectionEnd?: () => void
  // Whether to use scroll position in calculations (for scrollable containers)
  useScroll?: boolean
}

export function useBoxSelection(options: BoxSelectionOptions) {
  const isSelecting = ref(false)
  const selectionStart = ref({x: 0, y: 0})
  const selectionCurrent = ref({x: 0, y: 0})
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
    if (!options.containerRef.value) return {x: 0, y: 0}

    const containerRect = options.containerRef.value.getBoundingClientRect()

    let x = e.clientX - containerRect.left
    let y = e.clientY - containerRect.top

    if (options.useScroll) {
      x += options.containerRef.value.scrollLeft
      y += options.containerRef.value.scrollTop
    }

    return {x, y}
  }

  function handleMouseDown(e: MouseEvent) {
    if (e.button !== 0 || !options.containerRef.value) return

    // Don't start if clicking on scrollbar (simplified check, usually handled by browser but good to be safe)
    // Or if target is interactive? For now assume caller handles filters

    isDragOperation = false

    const {x, y} = getPointInContainer(e)
    selectionStart.value = {x, y}
    selectionCurrent.value = {x, y}

    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
  }

  function handleMouseMove(e: MouseEvent) {
    if (!isSelecting.value) {
      const {x, y} = getPointInContainer(e)
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

    const {x, y} = getPointInContainer(e)
    selectionCurrent.value = {x, y}
    // updateSelection()
  }

  function handleMouseUp() {
    window.removeEventListener('mousemove', handleMouseMove)
    window.removeEventListener('mouseup', handleMouseUp)

    if (isSelecting.value) {
      isSelecting.value = false
      isDragOperation = true
      // Reset flag on next tick to allow click handlers to know it was a drag
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

    options.itemRefs.forEach((el, index) => {
      const image = imageStore.images[index]
      if (!image) return
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
      if (!isIntersecting) return;

      const id = image.id;
      if (imageStore.selectedImages.has(id)) {
        imageStore.selectedImages.delete(id)
      } else {
        imageStore.selectedImages.add(id)
      }
    })
  }

  // Clean up in case component unmounts while dragging
  onUnmounted(() => {
    window.removeEventListener('mousemove', handleMouseMove)
    window.removeEventListener('mouseup', handleMouseUp)
  })

  return {
    selectionBoxStyle,
    handleMouseDown,
    // Expose check for drag operation so click handlers can ignore clicks after drag
    isDragOperation: () => isDragOperation
  }
}
