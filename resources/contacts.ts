import type { ContactRecord } from '#/db/schema.ts'
import type { ContactResource } from '#/generated/handlers/index.ts'

export function toContactResource(contact: ContactRecord): ContactResource {
  return {
    ...contact,
    id: contact.id.toString(),
  }
}
