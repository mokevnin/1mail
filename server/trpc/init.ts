import { initTRPC } from '@trpc/server'
import type { TrpcContext } from './context.ts'

export const t = initTRPC.context<TrpcContext>().create()
