import { bigserial, boolean, jsonb, pgTable, text, timestamp } from 'drizzle-orm/pg-core'

export const apiTokenScopes = [
  'contacts:read',
  'contacts:write',
  'segments:read',
  'segments:write',
  'broadcasts:read',
  'broadcasts:write',
  'tokens:manage',
] as const

export type ApiTokenScope = (typeof apiTokenScopes)[number]

export const users = pgTable('users', {
  id: bigserial('id', { mode: 'bigint' }).primaryKey(),
  name: text('name').notNull(),
  email: text('email').notNull().unique(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
})

export const contacts = pgTable('contacts', {
  id: bigserial('id', { mode: 'bigint' }).primaryKey(),
  email: text('email').notNull().unique(),
  firstName: text('first_name'),
  lastName: text('last_name'),
  status: text('status').$type<'active' | 'unsubscribed'>().notNull().default('active'),
  timeZone: text('time_zone'),
  customFields: jsonb('custom_fields').$type<Record<string, string>>(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true })
    .defaultNow()
    .notNull()
    .$onUpdate(() => new Date()),
})

export const apiTokens = pgTable('api_tokens', {
  id: bigserial('id', { mode: 'bigint' }).primaryKey(),
  name: text('name').notNull(),
  prefix: text('prefix').notNull().unique(),
  secretHash: text('secret_hash').notNull(),
  scopes: jsonb('scopes').$type<ApiTokenScope[]>().notNull(),
  expiresAt: timestamp('expires_at', { withTimezone: true }),
  revokedAt: timestamp('revoked_at', { withTimezone: true }),
  lastUsedAt: timestamp('last_used_at', { withTimezone: true }),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true })
    .defaultNow()
    .notNull()
    .$onUpdate(() => new Date()),
})

export const events = pgTable('events', {
  id: bigserial('id', { mode: 'bigint' }).primaryKey(),
  subjectId: text('subject_id').notNull(),
  email: text('email'),
  phone: text('phone'),
  action: text('action').notNull(),
  properties: jsonb('properties').$type<Record<string, unknown>>(),
  occurredAt: timestamp('occurred_at', { withTimezone: true }),
  prospect: boolean('prospect'),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
})

export type UserRecord = typeof users.$inferSelect
export type UserInsert = typeof users.$inferInsert
export type ContactRecord = typeof contacts.$inferSelect
export type ContactInsert = typeof contacts.$inferInsert
export type EventRecord = typeof events.$inferSelect
export type EventInsert = typeof events.$inferInsert
export type ApiTokenRecord = typeof apiTokens.$inferSelect
export type ApiTokenInsert = typeof apiTokens.$inferInsert
