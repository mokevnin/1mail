import type { FastifyRequest } from 'fastify'
import fp from 'fastify-plugin'

import type { AppDatabase } from '../db/runtime.ts'
import type { PaginationQuery } from '../lib/pagination.ts'
import { buildPage, resolvePagination } from '../lib/pagination.ts'

type CountSource = Parameters<AppDatabase['$count']>[0]

type PaginationBuilder<TItem> = {
  as(alias: string): unknown
  limit(limit: number): {
    offset(offset: number): Promise<TItem[]>
  }
}

type PaginateInput<TTable, TItem, TResult = TItem> = {
  table: TTable
  query: (table: TTable) => PaginationBuilder<TItem>
  map?: (item: TItem) => TResult
}

export default fp(
  async (fastify) => {
    fastify.decorateRequest('paginate', async function paginate<
      TTable,
      TItem,
      TResult = TItem,
    >(this: FastifyRequest, { table, query, map }: PaginateInput<TTable, TItem, TResult>) {
      const { page, pageSize, offset } = resolvePagination(
        this.query as PaginationQuery | undefined,
      )

      const buildQuery = () => query(table)

      const [totalItems, rows] = await Promise.all([
        fastify.db.$count(buildQuery().as('paginated_items') as CountSource),
        buildQuery().limit(pageSize).offset(offset),
      ])

      return buildPage({
        items: rows.map((row) => map?.(row) ?? (row as unknown as TResult)),
        page,
        pageSize,
        totalItems,
      })
    })
  },
  {
    name: 'paginate',
    dependencies: ['drizzle'],
  },
)
