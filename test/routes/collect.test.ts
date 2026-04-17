import { eq } from 'drizzle-orm'
import { expect, test } from 'vitest'
import { events, trackingProfiles, trackingVisitors } from '../../db/schema.ts'
import { build, collectHeaders, DefaultCollectSiteKey } from '../helper.ts'

test('collect identify creates profile and links visitor', async (t) => {
  const app = await build(t)

  const response = await app.inject({
    method: 'POST',
    url: '/collect/identify',
    headers: collectHeaders(),
    payload: {
      visitorId: 'visitor_1',
      email: 'Alice@example.com',
      traits: {
        plan: 'pro',
      },
    },
  })

  expect(response.statusCode).toBe(200)
  expect(response.json()).toEqual({ ok: true })

  const [profile] = await app.db
    .select()
    .from(trackingProfiles)
    .where(eq(trackingProfiles.email, 'alice@example.com'))
    .limit(1)
  const [visitor] = await app.db
    .select()
    .from(trackingVisitors)
    .where(eq(trackingVisitors.visitorId, 'visitor_1'))
    .limit(1)

  expect(profile?.subjectId).toBe('email:alice@example.com')
  expect(profile?.traits).toEqual({ plan: 'pro' })
  expect(visitor?.profileId).toBe(profile?.id)
})

test('collect events write anonymous events without prior identify', async (t) => {
  const app = await build(t)

  const response = await app.inject({
    method: 'POST',
    url: '/collect/events',
    headers: collectHeaders(),
    payload: {
      events: [
        {
          visitorId: 'anon_visitor',
          action: 'page.view',
          properties: {
            path: '/contacts',
          },
        },
      ],
    },
  })

  expect(response.statusCode).toBe(204)

  const [storedEvent] = await app.db
    .select()
    .from(events)
    .where(eq(events.action, 'page.view'))
    .limit(1)

  expect(storedEvent?.subjectId).toBe('visitor:anon_visitor')
  expect(storedEvent?.properties).toEqual({ path: '/contacts' })
})

test('collect events use identified profile for subsequent events', async (t) => {
  const app = await build(t)

  await app.inject({
    method: 'POST',
    url: '/collect/identify',
    headers: collectHeaders(),
    payload: {
      visitorId: 'visitor_2',
      email: 'bob@example.com',
    },
  })

  const response = await app.inject({
    method: 'POST',
    url: '/collect/events',
    headers: collectHeaders(),
    payload: {
      events: [
        {
          visitorId: 'visitor_2',
          action: 'contact.created',
          properties: {
            contactId: '42',
          },
        },
      ],
    },
  })

  expect(response.statusCode).toBe(204)

  const [storedEvent] = await app.db
    .select()
    .from(events)
    .where(eq(events.action, 'contact.created'))
    .limit(1)

  expect(storedEvent?.subjectId).toBe('email:bob@example.com')
  expect(storedEvent?.email).toBe('bob@example.com')
})

test('collect endpoints reject invalid collect key', async (t) => {
  const app = await build(t)

  const response = await app.inject({
    method: 'POST',
    url: '/collect/events',
    headers: collectHeaders(`${DefaultCollectSiteKey}-wrong`),
    payload: {
      events: [
        {
          visitorId: 'visitor_3',
          action: 'page.view',
        },
      ],
    },
  })

  expect(response.statusCode).toBe(401)
})
