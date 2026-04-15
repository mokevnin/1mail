import cors from '@fastify/cors'
import { fastifyTRPCPlugin } from '@trpc/server/adapters/fastify'
import fp from 'fastify-plugin'
import { router as appRouter } from '../src/trpc/trpc.ts'

export default fp(async (fastify) => {
  await fastify.register(cors, {
    origin: '*', // Adjust for production
  })

  await fastify.register(fastifyTRPCPlugin, {
    prefix: '/trpc',
    trpcOptions: {
      router: appRouter,
      createContext: () => ({}),
    },
  })
})
