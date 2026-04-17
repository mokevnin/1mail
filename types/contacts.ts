import type { z } from 'zod'

import type { zContactsCreateBodyInput, zContactsUpdateBodyInput } from '../lib/http-zod.ts'

export type CreateContactInput = z.output<typeof zContactsCreateBodyInput>
export type UpdateContactInput = z.output<typeof zContactsUpdateBodyInput>
