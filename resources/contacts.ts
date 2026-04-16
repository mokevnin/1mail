import type { ContactRecord } from '../db/schema.ts'
import type { ContactResource } from '../generated/handlers/index.ts'
import { toTimestamp } from '../lib/utils.ts'

export function toContactResource(contact: ContactRecord): ContactResource {
  return {
    ...contact,
    createdAt: toTimestamp(contact.createdAt),
    updatedAt: toTimestamp(contact.updatedAt),
  }
}
