import type { ApiTokenInfo } from '#/generated/handlers/index.ts'
import type { ApiTokenMetadata } from '#/types/api-tokens.ts'

export function toApiTokenInfo({ id, ...token }: ApiTokenMetadata): ApiTokenInfo {
  return {
    ...token,
    id: id.toString(),
  }
}
