import type { FastifyRequest } from 'fastify'
import fp from 'fastify-plugin'

import type { PaginationBuilder } from '../lib/pagination.ts'
import { paginateQuery } from '../lib/pagination.ts'

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
      return paginateQuery({
        db: fastify.db,
        pagination: this.query,
        table,
        query,
        map,
      })
    })
  },
  {
    name: 'paginate',
    dependencies: ['drizzle'],
  },
)
