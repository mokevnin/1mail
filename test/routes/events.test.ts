import { expect, test } from 'vitest'
import { authHeader, bootstrapApiToken, build } from '../helper.ts'

test('events batch write returns 204', async (t) => {
  const app = await build(t)
  const bootstrapResponse = await bootstrapApiToken(app)
  const { token } = bootstrapResponse.json()

  const response = await app.inject({
    method: 'POST',
    url: '/events',
    headers: authHeader(token),
    payload: {
      events: [
        {
          subjectId: 'user_1',
          email: 'alice@example.com',
          phone: '+15550001',
          action: 'emailOpened',
          prospect: true,
          properties: { campaign: 'welcome' },
          occurredAt: '2026-04-16T12:00:00.000Z',
        },
        {
          subjectId: 'user_2',
          action: 'emailClicked',
        },
      ],
    },
  })

  expect(response.statusCode).toBe(204)
})

test('events validate required fields', async (t) => {
  const app = await build(t)
  const bootstrapResponse = await bootstrapApiToken(app)
  const { token } = bootstrapResponse.json()

  const missingSubjectIdResponse = await app.inject({
    method: 'POST',
    url: '/events',
    headers: authHeader(token),
    payload: {
      events: [{ action: 'emailOpened' }],
    },
  })

  expect(missingSubjectIdResponse.statusCode).toBe(400)

  const missingActionResponse = await app.inject({
    method: 'POST',
    url: '/events',
    headers: authHeader(token),
    payload: {
      events: [{ subjectId: 'user_1' }],
    },
  })

  expect(missingActionResponse.statusCode).toBe(400)
})

test('event actions list returns 200 with pagination params', async (t) => {
  const app = await build(t)
  const bootstrapResponse = await bootstrapApiToken(app)
  const { token } = bootstrapResponse.json()

  await app.inject({
    method: 'POST',
    url: '/events',
    headers: authHeader(token),
    payload: {
      events: [
        { subjectId: 'user_1', action: 'bounced' },
        { subjectId: 'user_2', action: 'opened' },
        { subjectId: 'user_3', action: 'clicked' },
        { subjectId: 'user_4', action: 'opened' },
      ],
    },
  })

  const firstPageResponse = await app.inject({
    method: 'GET',
    url: '/event-actions?page=1&pageSize=2',
    headers: authHeader(token),
  })

  expect(firstPageResponse.statusCode).toBe(200)

  const secondPageResponse = await app.inject({
    method: 'GET',
    url: '/event-actions?page=2&pageSize=2',
    headers: authHeader(token),
  })

  expect(secondPageResponse.statusCode).toBe(200)
})
