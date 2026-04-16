import { TRPCError } from '@trpc/server'

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
