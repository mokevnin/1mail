import type { EventInsert } from '../db/schema.ts'

export type CollectIdentifyPayload = {
  visitorId: string
  email?: string | null
  phone?: string | null
  subjectId?: string | null
  traits?: Record<string, unknown> | null
}

export type CollectEventPayload = {
  visitorId: string
  action: string
  properties?: Record<string, unknown> | null
  occurredAt?: Date | null
}

export function assertCollectKey(value: string | string[] | undefined): void {
  const configured = process.env.COLLECT_SITE_KEY

  if (!configured) {
    throw Object.assign(new Error('Collect site key is not configured'), { statusCode: 503 })
  }

  if (Array.isArray(value)) {
    throw Object.assign(new Error('Collect key header must be provided exactly once'), {
      statusCode: 400,
    })
  }

  if (!value || value !== configured) {
    throw Object.assign(new Error('Invalid collect key'), { statusCode: 401 })
  }
}

export function normalizeVisitorId(value: string): string {
  const normalized = value.trim()

  if (!normalized) {
    throw Object.assign(new Error('visitorId is required'), { statusCode: 400 })
  }

  return normalized
}

export function normalizeEmail(value: string | null | undefined): string | null {
  const normalized = value?.trim().toLowerCase()

  return normalized ? normalized : null
}

export function normalizePhone(value: string | null | undefined): string | null {
  const normalized = value?.trim()

  return normalized ? normalized : null
}

export function normalizeSubjectId(value: string | null | undefined): string | null {
  const normalized = value?.trim()

  return normalized ? normalized : null
}

export function normalizeTraits(
  value: Record<string, unknown> | null | undefined,
): Record<string, unknown> {
  return value ?? {}
}

export function deriveCanonicalSubjectId(input: {
  email?: string | null
  phone?: string | null
  subjectId?: string | null
}): string {
  const subjectId = normalizeSubjectId(input.subjectId)

  if (subjectId) {
    return subjectId
  }

  const email = normalizeEmail(input.email)
  if (email) {
    return `email:${email}`
  }

  const phone = normalizePhone(input.phone)
  if (phone) {
    return `phone:${phone}`
  }

  throw Object.assign(new Error('identify requires subjectId, email, or phone'), {
    statusCode: 400,
  })
}

export function deriveAnonymousSubjectId(visitorId: string): string {
  return `visitor:${normalizeVisitorId(visitorId)}`
}

export function toCollectedEventInsert(input: {
  subjectId: string
  email?: string | null
  phone?: string | null
  action: string
  properties?: Record<string, unknown> | null
  occurredAt?: Date | null
}): Omit<EventInsert, 'id' | 'createdAt'> {
  return {
    subjectId: input.subjectId,
    email: input.email ?? null,
    phone: input.phone ?? null,
    action: input.action,
    properties: input.properties ?? null,
    occurredAt: input.occurredAt ?? null,
    prospect: null,
  }
}
