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

type TrpcGuard = {
  found<T>(value: T | null | undefined): asserts value is T
}

export const trpcGuard: TrpcGuard = {
  found<T>(value: T | null | undefined): asserts value is T {
    if (value == null) {
      throw new TRPCError({
        code: 'NOT_FOUND',
        message: 'Not Found',
      })
    }
  },
}
