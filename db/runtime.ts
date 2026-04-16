import type { NodePgDatabase } from 'drizzle-orm/node-postgres'
import type { PgliteDatabase } from 'drizzle-orm/pglite'
import type * as schema from './schema.ts'

export type DbDriver = 'pglite' | 'postgres'
export type AppDatabase = NodePgDatabase<typeof schema> | PgliteDatabase<typeof schema>

export function resolveDbDriver(): DbDriver {
  const configured = process.env.DB_DRIVER

  if (configured === 'postgres' || configured === 'pglite') {
    return configured
  }

  return process.env.NODE_ENV === 'production' ? 'postgres' : 'pglite'
}

export function isUniqueViolation(error: unknown): boolean {
  const isUniqueMessage = (message?: string) => {
    const normalized = message?.toLowerCase()

    return (
      normalized?.includes('unique') === true ||
      normalized?.includes('duplicate key') === true ||
      normalized?.includes('constraint') === true
    )
  }

  const walk = (current: unknown): boolean => {
    if (!current || typeof current !== 'object') {
      return false
    }

    const code = 'code' in current && typeof current.code === 'string' ? current.code : undefined
    const message =
      'message' in current && typeof current.message === 'string' ? current.message : undefined

    if (code === '23505' || isUniqueMessage(message)) {
      return true
    }

    return 'cause' in current ? walk(current.cause) : false
  }

  return walk(error)
}
