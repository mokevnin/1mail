import type { Queue } from '@platformatic/job-queue'
import 'fastify'
import type { AppDatabase } from '../db/runtime.ts'
import type { Page } from '../lib/pagination.ts'
import type { ApiTokenAuth } from './api-tokens.ts'

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
    found<T extends { id: bigint }>(value: T | null | undefined): asserts value is T
  }

  interface AppHelpers {
    guard: AppGuard
    throwHttpError(error: import('../types/errors.ts').UseCaseError): never
  }

  interface FastifyRequest {
    auth: ApiTokenAuth | null
    paginate<TTable, TItem, TResult = TItem>(
      input: PaginateInput<TTable, TItem, TResult>,
    ): Promise<Page<TResult>>
  }

  interface FastifyContextConfig {
    auth?: boolean
  }

  interface FastifyInstance {
    app: AppHelpers
    db: AppDatabase
    queue: Queue<Record<string, unknown>, { success: true }>
    requireAuth: import('@fastify/auth').FastifyAuthFunction
    someSupport(): string
    verifyApiToken: (
      request: FastifyRequest,
      reply: import('fastify').FastifyReply,
    ) => Promise<void>
  }
}
