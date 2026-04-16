import type { FastifyPluginAsync } from 'fastify'
import type { RouteHandlers } from '../../generated/handlers/fastify.gen.ts'

const handlers = {
  segmentsList: async (_request, _reply) => {
    throw new Error('segmentsList is not implemented')
  },
  segmentsCreate: async (_request, _reply) => {
    throw new Error('segmentsCreate is not implemented')
  },
  segmentsGet: async (_request, _reply) => {
    throw new Error('segmentsGet is not implemented')
  },
  segmentsUpdate: async (_request, _reply) => {
    throw new Error('segmentsUpdate is not implemented')
  },
  segmentsDelete: async (_request, _reply) => {
    throw new Error('segmentsDelete is not implemented')
  },
} satisfies Pick<
  RouteHandlers,
  'segmentsList' | 'segmentsCreate' | 'segmentsGet' | 'segmentsUpdate' | 'segmentsDelete'
>

const segments: FastifyPluginAsync = async (fastify, _opts): Promise<void> => {
  fastify.get('/segments', handlers.segmentsList)
  fastify.post('/segments', handlers.segmentsCreate)
  fastify.get('/segments/:id', handlers.segmentsGet)
  fastify.put('/segments/:id', handlers.segmentsUpdate)
  fastify.delete('/segments/:id', handlers.segmentsDelete)
}

export default segments
