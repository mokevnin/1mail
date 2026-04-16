import type { FastifyInstance } from 'fastify'
import type { AppDatabase } from '../../db/runtime.ts'

export type TrpcContext = {
  db: AppDatabase
}

export function createTrpcContext(fastify: FastifyInstance): TrpcContext {
  return {
    db: fastify.db,
  }
}
