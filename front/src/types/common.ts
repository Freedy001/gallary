export interface Pageable<T> {
  list: T[]
  total: number
  page: number
  page_size: number
  total_pages?: number
}

export const emptyPage = {total: 0, list: [], page: 0, page_size: 0} as Pageable<any>