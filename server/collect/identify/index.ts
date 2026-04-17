import type { FastifyInstance, FastifyPluginAsync } from 'fastify'
import { zCollectIdentifyCreateBody } from '#/generated/handlers/zod.gen.ts'
import { assertCollectKey } from '#/lib/collect.ts'
import { identifyVisitor } from '#/use-cases/collect.ts'

const collectIdentifyPlugin: FastifyPluginAsync = async (
  fastify: FastifyInstance,
  _opts,
): Promise<void> => {
  fastify.post(
    '/',
    {
      config: { auth: false },
      schema: {
        body: zCollectIdentifyCreateBody,
      },
    },
    async (request, reply) => {
      assertCollectKey(request.headers['x-collect-key'])
      await identifyVisitor(fastify.db, request.body as never)

      return reply.code(200).send({ ok: true as const })
    },
  )
}

export default collectIdentifyPlugin
