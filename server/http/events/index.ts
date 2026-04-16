import type { FastifyInstance, FastifyPluginAsync } from 'fastify'

import { events } from '../../../db/schema.ts'
import type { RouteHandlers } from '../../../generated/handlers/fastify.gen.ts'
import { toFastifySchema } from '../../../lib/openapi.ts'
import { toEventRecordInsert } from '../../../resources/events.ts'

const eventsPlugin: FastifyPluginAsync = async (fastify: FastifyInstance, _opts): Promise<void> => {
  const handlers = {
    eventsCreate: async (request, reply) => {
      const values = request.body.events.map(toEventRecordInsert)

      if (values.length > 0) {
        await fastify.db.insert(events).values(values)
      }

      return reply.code(204).send()
    },
  } satisfies Pick<RouteHandlers, 'eventsCreate'>

  fastify.post(
    '/',
    {
      schema: toFastifySchema('/events', 'POST'),
    },
    handlers.eventsCreate,
  )
}

export default eventsPlugin
