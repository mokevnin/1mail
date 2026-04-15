import { drizzle } from 'drizzle-orm/node-postgres'
import fp from 'fastify-plugin'
import pg from 'pg'
import * as schema from '../db/schema.ts'

export default fp(async (fastify) => {
  const connectionString = process.env.DATABASE_URL
  if (!connectionString) {
    throw new Error('DATABASE_URL is required')
  }
  const pool = new pg.Pool({ connectionString })

  const db = drizzle({ client: pool, schema })

  fastify.decorate('db', db)

  fastify.addHook('onClose', async () => {
    await pool.end()
  })
})
