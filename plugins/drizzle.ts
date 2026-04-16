import path from 'node:path'
import { PGlite } from '@electric-sql/pglite'
import { drizzle as drizzleNodePg } from 'drizzle-orm/node-postgres'
import { drizzle as drizzlePglite } from 'drizzle-orm/pglite'
import { migrate as migratePglite } from 'drizzle-orm/pglite/migrator'
import fp from 'fastify-plugin'
import pg from 'pg'
import { resolveDbDriver } from '../db/runtime.ts'
import * as schema from '../db/schema.ts'

export default fp(
  async (fastify) => {
    const driver = resolveDbDriver()

    if (driver === 'postgres') {
      const connectionString = process.env.DATABASE_URL
      if (!connectionString) {
        throw new Error('DATABASE_URL is required when DB_DRIVER=postgres')
      }

      const pool = new pg.Pool({ connectionString })
      const db = drizzleNodePg({ client: pool, schema })

      fastify.decorate('db', db)

      fastify.addHook('onClose', async () => {
        await pool.end()
      })

      return
    }

    const client = new PGlite()
    const db = drizzlePglite({ client, schema })

    await migratePglite(db, {
      migrationsFolder: path.join(import.meta.dirname, '..', 'drizzle'),
    })

    fastify.decorate('db', db)

    fastify.addHook('onClose', async () => {
      await client.close()
    })
  },
  {
    name: 'drizzle',
  },
)
