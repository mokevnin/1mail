import type { FastifyPluginAsync } from 'fastify'
import type { RouteHandlers } from '#/generated/handlers/fastify.gen.ts'

const handlers = {
  broadcastsList: async (_request, _reply) => {
    throw new Error('broadcastsList is not implemented')
  },
  broadcastsCreate: async (_request, _reply) => {
    throw new Error('broadcastsCreate is not implemented')
  },
  broadcastsGet: async (_request, _reply) => {
    throw new Error('broadcastsGet is not implemented')
  },
  broadcastsUpdate: async (_request, _reply) => {
    throw new Error('broadcastsUpdate is not implemented')
  },
  broadcastsDelete: async (_request, _reply) => {
    throw new Error('broadcastsDelete is not implemented')
  },
} satisfies Pick<
  RouteHandlers,
  'broadcastsList' | 'broadcastsCreate' | 'broadcastsGet' | 'broadcastsUpdate' | 'broadcastsDelete'
>

const broadcasts: FastifyPluginAsync = async (fastify, _opts): Promise<void> => {
  fastify.get('/broadcasts', handlers.broadcastsList)
  fastify.post('/broadcasts', handlers.broadcastsCreate)
  fastify.get('/broadcasts/:id', handlers.broadcastsGet)
  fastify.put('/broadcasts/:id', handlers.broadcastsUpdate)
  fastify.delete('/broadcasts/:id', handlers.broadcastsDelete)
}

export default broadcasts
