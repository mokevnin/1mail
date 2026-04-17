import type { z } from 'zod'

import type { zContactsCreateBody, zContactsUpdateBody } from '../generated/handlers/zod.gen.ts'

export type CreateContactInput = z.output<typeof zContactsCreateBody>
export type UpdateContactInput = z.output<typeof zContactsUpdateBody>
