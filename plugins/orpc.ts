import cors from '@fastify/cors'
import { RPCHandler } from '@orpc/server/fastify'
import type { FastifyReply, FastifyRequest } from 'fastify'
import fp from 'fastify-plugin'

import type { OrpcContext } from '../server/orpc/router.ts'

export default fp(
  async (fastify) => {
    await fastify.register(cors, {
      origin: '*',
    })

    let handler: RPCHandler<OrpcContext> | undefined

    async function getHandler() {
      if (!handler) {
        const { router } = await import('../server/orpc/router.ts')
        handler = new RPCHandler(router)
      }

      return handler
    }

    const handleOrpcRequest = async (request: FastifyRequest, reply: FastifyReply) => {
      const handler = await getHandler()

      const { matched } = await handler.handle(request, reply, {
        context: {
          db: fastify.db,
        },
        prefix: '/orpc',
      })

      if (!matched) {
        return reply.code(404).send('Not found')
      }
    }

    fastify.all('/orpc', handleOrpcRequest)
    fastify.all('/orpc/*', handleOrpcRequest)
  },
  {
    dependencies: ['drizzle'],
  },
)
