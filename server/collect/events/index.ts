import type { FastifyInstance, FastifyPluginAsync } from 'fastify'
import { assertCollectKey } from '#/lib/collect.ts'
import { zCollectEventsBodyInput } from '#/lib/http-zod.ts'
import { collectEvents } from '#/use-cases/collect.ts'

const collectEventsPlugin: FastifyPluginAsync = async (
  fastify: FastifyInstance,
  _opts,
): Promise<void> => {
  fastify.post(
    '/',
    {
      config: { auth: false },
      schema: {
        body: zCollectEventsBodyInput,
      },
    },
    async (request, reply) => {
      assertCollectKey(request.headers['x-collect-key'])
      await collectEvents(fastify.db, request.body as never)

      return reply.code(204).send()
    },
  )
}

export default collectEventsPlugin
