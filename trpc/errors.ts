import { TRPCError } from '@trpc/server'

import type { UseCaseError } from '../types/errors.ts'

const trpcErrorCodeByUseCaseCode = {
  conflict: 'CONFLICT',
  not_found: 'NOT_FOUND',
  internal: 'INTERNAL_SERVER_ERROR',
} as const

export function throwTrpcError(error: UseCaseError): never {
  throw new TRPCError({
    code: trpcErrorCodeByUseCaseCode[error.code],
    message: error.message,
  })
}
