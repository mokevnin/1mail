import cors from '@fastify/cors'
import { fastifyTRPCPlugin } from '@trpc/server/adapters/fastify'
import fp from 'fastify-plugin'
import { createTrpcContext } from '../server/trpc/context.ts'
import { appRouter } from '../server/trpc/routers/index.ts'

export default fp(async (fastify) => {
  await fastify.register(cors, {
    origin: '*', // Adjust for production
  })

  await fastify.register(fastifyTRPCPlugin, {
    prefix: '/trpc',
    trpcOptions: {
      router: appRouter,
      createContext: () => createTrpcContext(fastify),
    },
  })
})
