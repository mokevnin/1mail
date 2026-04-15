import { drizzle } from 'drizzle-orm/node-postgres'
import fp from 'fastify-plugin'
import pg from 'pg'
import * as schema from '../src/db/schema.ts'

declare module 'fastify' {
  interface FastifyInstance {
    db: ReturnType<typeof drizzle<typeof schema>>
  }
}

export default fp(async (fastify) => {
  const connectionString =
    process.env.DATABASE_URL || 'postgres://postgres:postgres@localhost:5432/1mail'
  const pool = new pg.Pool({ connectionString })

  const db = drizzle(pool, { schema })

  fastify.decorate('db', db)

  fastify.addHook('onClose', async () => {
    await pool.end()
  })
})
