declare module 'vue-virtual-scroller' {
  import type {DefineComponent} from 'vue'

  export interface RecycleScrollerProps {
    items: any[]
    itemSize?: number
    minItemSize?: number
    keyField?: string
    pageMode?: boolean
    buffer?: number
    emitUpdate?: boolean
  }

  export interface DynamicScrollerProps {
    items: any[]
    minItemSize: number
    keyField?: string
    pageMode?: boolean
    buffer?: number
    emitUpdate?: boolean
  }

  export interface DynamicScrollerItemProps {
    item: any
    active: boolean
    sizeDependencies?: any[]
    watchData?: boolean
    tag?: string
    emitResize?: boolean
    onResize?: () => void
    dataIndex?: number
  }

  export const RecycleScroller: DefineComponent<RecycleScrollerProps>
  export const DynamicScroller: DefineComponent<DynamicScrollerProps>
  export const DynamicScrollerItem: DefineComponent<DynamicScrollerItemProps>
}
