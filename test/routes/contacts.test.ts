import { eq } from 'drizzle-orm'
import { expect, test } from 'vitest'
import { contacts } from '../../db/schema.ts'
import { authHeader, bootstrapApiToken, build } from '../helper.ts'

test('contacts CRUD works with pglite', async (t) => {
  const app = await build(t)
  const bootstrapResponse = await bootstrapApiToken(app)
  const { token } = bootstrapResponse.json()

  const createResponse = await app.inject({
    method: 'POST',
    url: '/contacts',
    headers: authHeader(token),
    payload: {
      email: 'alice@example.com',
      firstName: 'Alice',
      lastName: 'Ng',
      timeZone: 'America/New_York',
      customFields: {
        company: 'Acme',
      },
    },
  })

  expect(createResponse.statusCode).toBe(201)

  const getResponse = await app.inject({
    method: 'GET',
    url: '/contacts/1',
    headers: authHeader(token),
  })

  expect(getResponse.statusCode).toBe(200)

  const updateResponse = await app.inject({
    method: 'PUT',
    url: '/contacts/1',
    headers: authHeader(token),
    payload: {
      firstName: 'Alicia',
      lastName: null,
      timeZone: null,
      customFields: {
        plan: 'pro',
      },
    },
  })

  expect(updateResponse.statusCode).toBe(200)

  const deleteResponse = await app.inject({
    method: 'DELETE',
    url: '/contacts/1',
    headers: authHeader(token),
  })

  expect(deleteResponse.statusCode).toBe(204)

  const missingResponse = await app.inject({
    method: 'GET',
    url: '/contacts/1',
    headers: authHeader(token),
  })

  expect(missingResponse.statusCode).toBe(404)
})

test('contacts reject duplicate email', async (t) => {
  const app = await build(t)
  const bootstrapResponse = await bootstrapApiToken(app)
  const { token } = bootstrapResponse.json()

  await app.inject({
    method: 'POST',
    url: '/contacts',
    headers: authHeader(token),
    payload: {
      email: 'duplicate@example.com',
    },
  })

  const duplicateResponse = await app.inject({
    method: 'POST',
    url: '/contacts',
    headers: authHeader(token),
    payload: {
      email: 'duplicate@example.com',
    },
  })

  expect(duplicateResponse.statusCode).toBe(409)
})

test('contacts list supports pagination and status filter', async (t) => {
  const app = await build(t)
  const bootstrapResponse = await bootstrapApiToken(app)
  const { token } = bootstrapResponse.json()

  await app.inject({
    method: 'POST',
    url: '/contacts',
    headers: authHeader(token),
    payload: {
      email: 'first@example.com',
    },
  })

  const secondCreateResponse = await app.inject({
    method: 'POST',
    url: '/contacts',
    headers: authHeader(token),
    payload: {
      email: 'second@example.com',
    },
  })

  expect(secondCreateResponse.statusCode).toBe(201)

  await app.db
    .update(contacts)
    .set({ status: 'unsubscribed' })
    .where(eq(contacts.email, 'second@example.com'))

  const filteredResponse = await app.inject({
    method: 'GET',
    url: '/contacts?status=unsubscribed&page=1&pageSize=1',
    headers: authHeader(token),
  })

  expect(filteredResponse.statusCode).toBe(200)
})
