import { createTRPCReact } from '@trpc/react-query'
import type { AppRouter } from '../types/trpc.ts'

export const trpc: ReturnType<typeof createTRPCReact<AppRouter>> = createTRPCReact<AppRouter>()
