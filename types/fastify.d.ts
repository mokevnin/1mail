import type { Queue } from '@platformatic/job-queue'
import 'fastify'
import type { AppDatabase } from '../db/runtime.ts'
import type { Page } from '../lib/pagination.ts'

interface PaginationBuilder<TItem> {
  as(alias: string): unknown
  limit(limit: number): {
    offset(offset: number): Promise<TItem[]>
  }
}

interface PaginateInput<TTable, TItem, TResult = TItem> {
  table: TTable
  query: (table: TTable) => PaginationBuilder<TItem>
  map?: (item: TItem) => TResult
}

declare module 'fastify' {
  interface AppGuard {
    found<T extends { id: number }>(value: T | null | undefined): asserts value is T
  }

  interface FastifyRequest {
    paginate<TTable, TItem, TResult = TItem>(
      input: PaginateInput<TTable, TItem, TResult>,
    ): Promise<Page<TResult>>
  }

  interface FastifyInstance {
    db: AppDatabase
    guard: AppGuard
    queue: Queue<Record<string, unknown>, { success: true }>
    someSupport(): string
  }
}
