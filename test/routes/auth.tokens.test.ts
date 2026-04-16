import { expect, test } from 'vitest'
import { authHeader, bootstrapApiToken, build, DefaultBootstrapToken } from '../helper.ts'

test('auth tokens bootstrap requires bootstrap header', async (t) => {
  const app = await build(t)

  const missingHeaderResponse = await app.inject({
    method: 'POST',
    url: '/auth/tokens/bootstrap',
    payload: {
      name: 'Bootstrap token',
      scopes: ['contacts:read'],
      expiresAt: null,
    },
  })

  expect(missingHeaderResponse.statusCode).toBe(401)

  const createResponse = await app.inject({
    method: 'POST',
    url: '/auth/tokens/bootstrap',
    headers: {
      'x-bootstrap-token': DefaultBootstrapToken,
    },
    payload: {
      name: 'Bootstrap token',
      scopes: ['contacts:read', 'tokens:manage'],
      expiresAt: null,
    },
  })

  expect(createResponse.statusCode).toBe(201)
})

test('auth tokens require bearer auth for protected endpoints', async (t) => {
  const app = await build(t)

  const response = await app.inject({
    method: 'GET',
    url: '/auth/tokens',
  })

  expect(response.statusCode).toBe(401)
})

test('auth tokens lifecycle works over http', async (t) => {
  const app = await build(t)
  const bootstrapResponse = await bootstrapApiToken(app, {
    name: 'Manager token',
    scopes: ['contacts:read', 'contacts:write', 'tokens:manage'],
  })
  const { token } = bootstrapResponse.json()

  const createResponse = await app.inject({
    method: 'POST',
    url: '/auth/tokens',
    headers: authHeader(token),
    payload: {
      name: 'Read token',
      scopes: ['contacts:read'],
      expiresAt: null,
    },
  })

  expect(createResponse.statusCode).toBe(201)

  const listResponse = await app.inject({
    method: 'GET',
    url: '/auth/tokens',
    headers: authHeader(token),
  })

  expect(listResponse.statusCode).toBe(200)

  const createdToken = listResponse
    .json()
    .items.find((item: { name: string }) => item.name === 'Read token')

  expect(createdToken).toBeTruthy()

  const meResponse = await app.inject({
    method: 'GET',
    url: '/auth/me',
    headers: authHeader(createResponse.json().token),
  })

  expect(meResponse.statusCode).toBe(200)
  expect(meResponse.json().name).toBe('Read token')

  const revokeResponse = await app.inject({
    method: 'DELETE',
    url: `/auth/tokens/${createdToken.id}`,
    headers: authHeader(token),
  })

  expect(revokeResponse.statusCode).toBe(204)
})

test('auth tokens enforce scopes', async (t) => {
  const app = await build(t)
  const bootstrapResponse = await bootstrapApiToken(app, {
    name: 'Read token',
    scopes: ['contacts:read'],
  })
  const { token } = bootstrapResponse.json()

  const response = await app.inject({
    method: 'POST',
    url: '/contacts',
    headers: authHeader(token),
    payload: {
      email: 'no-write@example.com',
    },
  })

  expect(response.statusCode).toBe(403)
})
