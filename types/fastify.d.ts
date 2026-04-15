import type { Queue } from '@platformatic/job-queue'
import type { drizzle } from 'drizzle-orm/node-postgres'
import 'fastify'
import type * as schema from '../db/schema.ts'

declare module 'fastify' {
  interface FastifyInstance {
    db: ReturnType<typeof drizzle<typeof schema>>
    queue: Queue<Record<string, unknown>, { success: true }>
    someSupport(): string
  }
}
