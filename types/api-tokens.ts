import type { ApiTokenRecord, ApiTokenScope } from '../db/schema.ts'

export type ApiTokenAuth = {
  tokenId: bigint
  name: string
  scopes: ApiTokenScope[]
}

export type ApiTokenMetadata = Omit<ApiTokenRecord, 'secretHash'>

export type CreateApiTokenInput = {
  name: string
  scopes: ApiTokenScope[]
  expiresAt?: Date | null
}

export type CreatedApiToken = {
  token: string
  record: ApiTokenMetadata
}
