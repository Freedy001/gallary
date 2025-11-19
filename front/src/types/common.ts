export interface Pageable<T> {
  list: T[]
  total: number
  page: number
  page_size: number
  total_pages: number
}