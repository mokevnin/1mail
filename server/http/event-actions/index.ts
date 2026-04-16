import { asc } from 'drizzle-orm'
import type { FastifyInstance, FastifyPluginAsync } from 'fastify'

import { events } from '../../../db/schema.ts'
import type { RouteHandlers } from '../../../generated/handlers/fastify.gen.ts'
import { toFastifySchema } from '../../../lib/openapi.ts'
import { toEventActionResource } from '../../../resources/events.ts'

const eventActionsPlugin: FastifyPluginAsync = async (
  fastify: FastifyInstance,
  _opts,
): Promise<void> => {
  const handlers = {
    eventActionsList: async (request, reply) => {
      const response = await request.paginate({
        table: events,
        query: (table) =>
          fastify.db
            .selectDistinct({ action: table.action })
            .from(table)
            .orderBy(asc(table.action)),
        map: (row) => toEventActionResource(row.action),
      })

      return reply.code(200).send(response)
    },
  } satisfies Pick<RouteHandlers, 'eventActionsList'>

  fastify.get(
    '/',
    {
      schema: toFastifySchema('/event-actions', 'GET'),
    },
    handlers.eventActionsList,
  )
}

export default eventActionsPlugin
