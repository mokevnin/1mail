import type { EventInsert } from '../db/schema.ts'
import type { EventActionResource } from '../generated/handlers/index.ts'

export function toEventRecordInsert(event: {
  subjectId: string
  action: string
  email?: string | null
  phone?: string | null
  prospect?: boolean | null
  properties?: Record<string, unknown> | null
  occurredAt?: string | null
}): Omit<EventInsert, 'id' | 'createdAt'> {
  return {
    subjectId: event.subjectId,
    action: event.action,
    email: event.email ?? null,
    phone: event.phone ?? null,
    prospect: event.prospect ?? null,
    properties: event.properties ?? null,
    occurredAt: event.occurredAt ? new Date(event.occurredAt) : null,
  }
}

export function toEventActionResource(action: string): EventActionResource {
  return { action }
}
