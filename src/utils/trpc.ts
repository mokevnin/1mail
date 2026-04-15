import { createTRPCReact } from '@trpc/react-query'
import type { AppRouter } from '../trpc/trpc.ts'

export const trpc = createTRPCReact<AppRouter>()
