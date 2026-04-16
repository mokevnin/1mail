import fp from 'fastify-plugin'
import { EntityNotFoundError } from '../lib/errors.ts'

export default fp(async (fastify) => {
  fastify.decorate('guard', {
    found<T extends { id: number }>(value: T | null | undefined): asserts value is T {
      if (value == null) {
        throw new EntityNotFoundError()
      }
    },
  })

  fastify.setErrorHandler((error, _request, reply) => {
    if (error instanceof EntityNotFoundError) {
      return reply.code(404).send({
        statusCode: 404,
        error: 'Not Found',
        message: error.message,
      })
    }

    return reply.send(error)
  })
})
