import { eq } from 'drizzle-orm'
import type { AppDatabase } from '../db/runtime.ts'
import { events, trackingProfiles, trackingVisitors } from '../db/schema.ts'
import {
  type CollectEventPayload,
  type CollectIdentifyPayload,
  deriveAnonymousSubjectId,
  deriveCanonicalSubjectId,
  normalizeEmail,
  normalizePhone,
  normalizeSubjectId,
  normalizeTraits,
  normalizeVisitorId,
  toCollectedEventInsert,
} from '../lib/collect.ts'

type TrackingResolution = {
  subjectId: string
  email: string | null
  phone: string | null
}

async function findProfileBySubjectId(db: AppDatabase, subjectId: string | null) {
  if (!subjectId) {
    return null
  }

  const [record] = await db
    .select()
    .from(trackingProfiles)
    .where(eq(trackingProfiles.subjectId, subjectId))
    .limit(1)

  return record ?? null
}

async function findProfileByEmail(db: AppDatabase, email: string | null) {
  if (!email) {
    return null
  }

  const [record] = await db
    .select()
    .from(trackingProfiles)
    .where(eq(trackingProfiles.email, email))
    .limit(1)

  return record ?? null
}

async function findProfileByPhone(db: AppDatabase, phone: string | null) {
  if (!phone) {
    return null
  }

  const [record] = await db
    .select()
    .from(trackingProfiles)
    .where(eq(trackingProfiles.phone, phone))
    .limit(1)

  return record ?? null
}

async function findOrCreateVisitor(db: AppDatabase, visitorId: string) {
  const [existing] = await db
    .select()
    .from(trackingVisitors)
    .where(eq(trackingVisitors.visitorId, visitorId))
    .limit(1)

  if (existing) {
    const [updated] = await db
      .update(trackingVisitors)
      .set({ lastSeenAt: new Date() })
      .where(eq(trackingVisitors.id, existing.id))
      .returning()

    return updated ?? existing
  }

  const [created] = await db
    .insert(trackingVisitors)
    .values({
      visitorId,
      lastSeenAt: new Date(),
    })
    .returning()

  if (!created) {
    throw new Error('Failed to create tracking visitor')
  }

  return created
}

async function upsertProfile(db: AppDatabase, input: CollectIdentifyPayload) {
  const subjectId = normalizeSubjectId(input.subjectId)
  const email = normalizeEmail(input.email)
  const phone = normalizePhone(input.phone)
  const traits = normalizeTraits(input.traits)
  const canonicalSubjectId = deriveCanonicalSubjectId({ subjectId, email, phone })

  const existing =
    (await findProfileBySubjectId(db, subjectId)) ??
    (await findProfileByEmail(db, email)) ??
    (await findProfileByPhone(db, phone))

  if (existing) {
    const [updated] = await db
      .update(trackingProfiles)
      .set({
        email: email ?? existing.email,
        phone: phone ?? existing.phone,
        traits: { ...(existing.traits ?? {}), ...traits },
      })
      .where(eq(trackingProfiles.id, existing.id))
      .returning()

    return updated ?? existing
  }

  const [created] = await db
    .insert(trackingProfiles)
    .values({
      subjectId: canonicalSubjectId,
      email,
      phone,
      traits,
    })
    .returning()

  if (!created) {
    throw new Error('Failed to create tracking profile')
  }

  return created
}

async function resolveTrackingIdentity(
  db: AppDatabase,
  visitorId: string,
): Promise<TrackingResolution> {
  const visitor = await findOrCreateVisitor(db, visitorId)

  if (!visitor.profileId) {
    return {
      subjectId: deriveAnonymousSubjectId(visitorId),
      email: null,
      phone: null,
    }
  }

  const [profile] = await db
    .select()
    .from(trackingProfiles)
    .where(eq(trackingProfiles.id, visitor.profileId))
    .limit(1)

  if (!profile) {
    return {
      subjectId: deriveAnonymousSubjectId(visitorId),
      email: null,
      phone: null,
    }
  }

  return {
    subjectId: profile.subjectId,
    email: profile.email ?? null,
    phone: profile.phone ?? null,
  }
}

export async function identifyVisitor(
  db: AppDatabase,
  input: CollectIdentifyPayload,
): Promise<void> {
  const visitorId = normalizeVisitorId(input.visitorId)
  const profile = await upsertProfile(db, input)
  const visitor = await findOrCreateVisitor(db, visitorId)

  await db
    .update(trackingVisitors)
    .set({
      profileId: profile.id,
      lastSeenAt: new Date(),
    })
    .where(eq(trackingVisitors.id, visitor.id))
}

export async function collectEvents(
  db: AppDatabase,
  input: { events: CollectEventPayload[] },
): Promise<void> {
  const values = []

  for (const event of input.events) {
    const visitorId = normalizeVisitorId(event.visitorId)
    const resolution = await resolveTrackingIdentity(db, visitorId)

    values.push(
      toCollectedEventInsert({
        subjectId: resolution.subjectId,
        email: resolution.email,
        phone: resolution.phone,
        action: event.action,
        properties: event.properties ?? null,
        occurredAt: event.occurredAt ?? null,
      }),
    )
  }

  if (values.length > 0) {
    await db.insert(events).values(values)
  }
}
