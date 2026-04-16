import { createInsertSchema } from 'drizzle-zod'
import { z } from 'zod'
import { users } from './schema.ts'

export const insertUserSchema = createInsertSchema(users, {
  email: (schema) => schema.check(z.email('Invalid email address')),
  name: (schema) => schema.min(1),
})
