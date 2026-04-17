import { eq } from 'drizzle-orm'
import type { AppDatabase } from '../db/runtime.ts'
import {
  type ApiTokenInsert,
  type ApiTokenRecord,
  type ApiTokenScope,
  apiTokens,
} from '../db/schema.ts'
import {
  createTokenValue,
  generateTokenPrefix,
  generateTokenSecret,
  hashTokenSecret,
  parseApiToken,
  verifyTokenSecret,
} from '../lib/api-tokens.ts'
import type {
  ApiTokenAuth,
  ApiTokenMetadata,
  CreateApiTokenInput,
  CreatedApiToken,
} from '../types/api-tokens.ts'

export function toApiTokenMetadata(record: ApiTokenRecord): ApiTokenMetadata {
  const { secretHash: _secretHash, ...metadata } = record

  return metadata
}

function normalizeScopes(scopes: ApiTokenScope[]): ApiTokenScope[] {
  return [...new Set(scopes)].sort()
}

export async function createApiToken(
  db: AppDatabase,
  input: CreateApiTokenInput,
): Promise<CreatedApiToken> {
  const prefix = generateTokenPrefix()
  const secret = generateTokenSecret()
  const secretHash = await hashTokenSecret(secret)

  const values: ApiTokenInsert = {
    name: input.name,
    prefix,
    secretHash,
    scopes: normalizeScopes(input.scopes),
    expiresAt: input.expiresAt ?? null,
  }

  const [created] = await db.insert(apiTokens).values(values).returning()
  if (!created) {
    throw new Error('Failed to create API token')
  }

  return {
    token: createTokenValue(prefix, secret),
    record: toApiTokenMetadata(created),
  }
}

export async function listApiTokens(db: AppDatabase): Promise<ApiTokenMetadata[]> {
  const records = await db.select().from(apiTokens)

  return records
    .map(toApiTokenMetadata)
    .sort((left, right) => right.createdAt.getTime() - left.createdAt.getTime())
}

export async function revokeApiToken(
  db: AppDatabase,
  id: bigint,
): Promise<ApiTokenMetadata | null> {
  const [revoked] = await db
    .update(apiTokens)
    .set({ revokedAt: new Date() })
    .where(eq(apiTokens.id, id))
    .returning()

  return revoked ? toApiTokenMetadata(revoked) : null
}

export async function getApiTokenById(
  db: AppDatabase,
  id: bigint,
): Promise<ApiTokenMetadata | null> {
  const [record] = await db.select().from(apiTokens).where(eq(apiTokens.id, id)).limit(1)

  return record ? toApiTokenMetadata(record) : null
}

export async function verifyApiToken(
  db: AppDatabase,
  token: string,
): Promise<ApiTokenMetadata | null> {
  const parsed = parseApiToken(token)
  if (!parsed) {
    return null
  }

  const [record] = await db
    .select()
    .from(apiTokens)
    .where(eq(apiTokens.prefix, parsed.prefix))
    .limit(1)
  if (!record) {
    return null
  }

  if (record.revokedAt) {
    return null
  }

  if (record.expiresAt && record.expiresAt.getTime() <= Date.now()) {
    return null
  }

  const secretMatches = await verifyTokenSecret(parsed.secret, record.secretHash)

  if (!secretMatches) {
    return null
  }

  const [updated] = await db
    .update(apiTokens)
    .set({ lastUsedAt: new Date() })
    .where(eq(apiTokens.id, record.id))
    .returning()

  return toApiTokenMetadata(updated ?? record)
}

export function requireScope(auth: ApiTokenAuth | null | undefined, scope: ApiTokenScope): void {
  if (!auth) {
    throw Object.assign(new Error('Authentication is required'), { statusCode: 401 })
  }

  if (!auth.scopes.includes(scope)) {
    throw Object.assign(new Error(`Missing required scope: ${scope}`), { statusCode: 403 })
  }
}
