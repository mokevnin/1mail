import fp from 'fastify-plugin'
import { EntityNotFoundError } from '../lib/errors.ts'
import type { UseCaseError } from '../types/errors.ts'

export default fp(async (fastify) => {
  fastify.decorate('app', {
    guard: {
      found<T extends { id: bigint }>(value: T | null | undefined): asserts value is T {
        if (value == null) {
          throw new EntityNotFoundError()
        }
      },
    },
    throwHttpError(error: UseCaseError): never {
      switch (error.code) {
        case 'conflict':
          throw fastify.httpErrors.conflict(error.message)
        case 'not_found':
          throw fastify.httpErrors.notFound(error.message)
        case 'internal':
          throw fastify.httpErrors.internalServerError(error.message)
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
