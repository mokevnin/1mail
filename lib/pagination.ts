const DEFAULT_PAGE = 1
const DEFAULT_PAGE_SIZE = 25

type PaginationQuery = {
  page?: number
  pageSize?: number
}

type Page<T> = {
  items: T[]
  page: number
  pageSize: number
  totalItems: number
  totalPages: number
}

type BuildPageInput<T> = {
  items: T[]
  page: number
  pageSize: number
  totalItems: number
}

export function resolvePagination(query?: PaginationQuery) {
  const page = query?.page ?? DEFAULT_PAGE
  const pageSize = query?.pageSize ?? DEFAULT_PAGE_SIZE

  return {
    page,
    pageSize,
    offset: (page - 1) * pageSize,
  }
}

export function buildPage<T>({ items, page, pageSize, totalItems }: BuildPageInput<T>): Page<T> {
  return {
    items,
    page,
    pageSize,
    totalItems,
    totalPages: totalItems === 0 ? 0 : Math.ceil(totalItems / pageSize),
  }
}

export { DEFAULT_PAGE, DEFAULT_PAGE_SIZE }
export type { Page, PaginationQuery }
