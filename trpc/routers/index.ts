import { t } from '../init.ts'
import { contactsRouter } from './contacts/router.ts'

export const appRouter = t.router({
  contacts: contactsRouter,
})

export type AppRouter = typeof appRouter
