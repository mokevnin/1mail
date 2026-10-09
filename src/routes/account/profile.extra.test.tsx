import { expect, test } from 'vitest'

import type { SiteUserResource } from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientFetch, requestOf } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { ProfilePage } from './profile.tsx'

const user: SiteUserResource = {
  id: '1',
  name: 'John',
  email: 'info@1mail.com',
  emailVerified: false,
  createdAt: '2026-01-01T00:00:00Z',
}

type Call = { method: string; path: string; body: unknown }

// Serves the profile endpoints and records every mutating call.
function serveProfile(
  calls: Call[],
  overrides: Record<string, (req: Request) => Response> = {},
  me: SiteUserResource = user,
) {
  mockClientFetch(async (input, init) => {
    const req = requestOf(input, init)
    const path = new URL(req.url).pathname
    const key = `${req.method} ${path.replace(/^.*\/(me.*)$/, '/$1')}`
    const text = await req.clone().text()
    if (req.method !== 'GET') {
      calls.push({ method: req.method, path, body: text ? JSON.parse(text) : null })
    }
    const override = overrides[key]
    if (override) return override(req)
    if (key === 'GET /me') return jsonResponse(me)
    return new Response(null, { status: 204 })
  })
}

test('an unverified user can ask for the verification email again', async () => {
  const calls: Call[] = []
  serveProfile(calls)
  const { screen } = await renderWithRouter(<ProfilePage />)

  await expect.element(screen.getByText('Not verified')).toBeInTheDocument()
  await screen.getByRole('button', { name: 'Resend verification email' }).click()

  await expect.element(screen.getByText('Verification email sent')).toBeInTheDocument()
  expect(calls.map((c) => `${c.method} ${c.path.replace(/^.*\/me/, '/me')}`)).toEqual([
    'POST /me/verification-email',
  ])
})

test('a verified user sees the verified badge and no resend prompt', async () => {
  serveProfile([], {}, { ...user, emailVerified: true })
  const { screen } = await renderWithRouter(<ProfilePage />)

  await expect.element(screen.getByText('Verified', { exact: true })).toBeInTheDocument()
  expect(screen.getByText('Your email address is not verified.').elements()).toHaveLength(0)
})

test('reports a failed verification resend', async () => {
  serveProfile([], {
    'POST /me/verification-email': () =>
      jsonResponse({ status: 429, detail: 'slow down' }, { status: 429 }),
  })
  const { screen } = await renderWithRouter(<ProfilePage />)

  await screen.getByRole('button', { name: 'Resend verification email' }).click()

  await expect.element(screen.getByText('Could not send verification email')).toBeInTheDocument()
})

test('changing the password sends the current and new password', async () => {
  const calls: Call[] = []
  serveProfile(calls, { 'PUT /me': () => jsonResponse(user) })
  const { screen } = await renderWithRouter(<ProfilePage />)

  await expect.element(screen.getByLabelText(/^Name/)).toHaveValue('John')
  await screen.getByLabelText('Current password', { exact: true }).first().fill('old-pass')
  await screen.getByLabelText(/^New password/).fill('new-pass-123')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.element(screen.getByText('Profile updated')).toBeInTheDocument()
  expect(calls.find((c) => c.method === 'PUT')?.body).toEqual({
    name: 'John',
    currentPassword: 'old-pass',
    newPassword: 'new-pass-123',
  })
})

test('reports a failed profile update', async () => {
  serveProfile([], {
    'PUT /me': () => jsonResponse({ status: 400, detail: 'wrong password' }, { status: 400 }),
  })
  const { screen } = await renderWithRouter(<ProfilePage />)

  await expect.element(screen.getByLabelText(/^Name/)).toHaveValue('John')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.element(screen.getByText('Failed to update profile')).toBeInTheDocument()
})

test('requests an email change and clears the form', async () => {
  const calls: Call[] = []
  serveProfile(calls)
  const { screen } = await renderWithRouter(<ProfilePage />)

  await screen.getByLabelText(/^New email/).fill(' next@1mail.com ')
  await screen
    .getByLabelText(/^Current password/)
    .last()
    .fill('pw-123456')
  await screen.getByRole('button', { name: 'Send confirmation link' }).click()

  await expect
    .element(screen.getByText('Check your new inbox for a confirmation link.'))
    .toBeInTheDocument()
  expect(calls.find((c) => c.path.endsWith('/email-change'))?.body).toEqual({
    newEmail: 'next@1mail.com',
    currentPassword: 'pw-123456',
  })
  await expect.element(screen.getByLabelText(/^New email/)).toHaveValue('')
})

test('reports a failed email change', async () => {
  serveProfile([], {
    'POST /me/email-change': () => jsonResponse({ status: 409, detail: 'taken' }, { status: 409 }),
  })
  const { screen } = await renderWithRouter(<ProfilePage />)

  await screen.getByLabelText(/^New email/).fill('next@1mail.com')
  await screen
    .getByLabelText(/^Current password/)
    .last()
    .fill('pw-123456')
  await screen.getByRole('button', { name: 'Send confirmation link' }).click()

  await expect.element(screen.getByText('Could not change email')).toBeInTheDocument()
})

test('shows an error when the profile fails to load', async () => {
  mockClientFetch(() => jsonResponse({ status: 500, detail: 'boom' }, { status: 500 }))
  const { screen } = await renderWithRouter(<ProfilePage />)

  await expect.element(screen.getByText('Failed to load profile').first()).toBeInTheDocument()
})
