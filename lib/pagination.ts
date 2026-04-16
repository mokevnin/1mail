import { clamp, get, isNumber, isObject } from 'es-toolkit/compat'

import type { AppDatabase } from '../db/runtime.ts'

const DEFAULT_PAGE = 1
const DEFAULT_PAGE_SIZE = 25

type PaginationQuery = {
  page?: number | undefined
  pageSize?: number | undefined
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

type CountSource = Parameters<AppDatabase['$count']>[0]

type PaginationBuilder<TItem> = {
  as(alias: string): unknown
  limit(limit: number): {
    offset(offset: number): Promise<TItem[]>
  }
}

type PaginateQueryInput<TTable, TItem, TResult = TItem> = {
  db: AppDatabase
  pagination: unknown
  table: TTable
  query: (table: TTable) => PaginationBuilder<TItem>
  map: ((item: TItem) => TResult) | undefined
}

function normalizePositiveInt(value: unknown, fallback: number) {
  if (!isNumber(value)) {
    return fallback
  }

  return clamp(value, 1, Number.MAX_SAFE_INTEGER)
}

export function resolvePagination(query?: unknown) {
  const page = normalizePositiveInt(isObject(query) ? get(query, 'page') : undefined, DEFAULT_PAGE)
  const pageSize = normalizePositiveInt(
    isObject(query) ? get(query, 'pageSize') : undefined,
    DEFAULT_PAGE_SIZE,
  )

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

export async function paginateQuery<TTable, TItem, TResult = TItem>({
  db,
  pagination,
  table,
  query,
  map,
}: PaginateQueryInput<TTable, TItem, TResult>): Promise<Page<TResult>> {
  const { page, pageSize, offset } = resolvePagination(pagination)
  const buildQuery = () => query(table)

  const [totalItems, rows] = await Promise.all([
    db.$count(buildQuery().as('paginated_items') as CountSource),
    buildQuery().limit(pageSize).offset(offset),
  ])

  return buildPage({
    items: rows.map((row) => map?.(row) ?? (row as unknown as TResult)),
    page,
    pageSize,
    totalItems,
  })
}

export type { Page, PaginationBuilder, PaginationQuery }
export { DEFAULT_PAGE, DEFAULT_PAGE_SIZE }
