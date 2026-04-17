import { initTRPC } from '@trpc/server'
import type { FastifyInstance } from 'fastify'
import superjson from 'superjson'

import type { AppDatabase } from '../db/runtime.ts'
import { createContactsRouter } from '../server/trpc/contacts/index.ts'

export type TrpcContext = {
  db: AppDatabase
}

export function createTrpcContext(fastify: FastifyInstance): TrpcContext {
  return {
    db: fastify.db,
  }
}

export const t = initTRPC.context<TrpcContext>().create({
  transformer: superjson,
})

const contactsRouter = createContactsRouter({ t })

export const appRouter = t.router({
  contacts: contactsRouter,
})

export const router = appRouter
