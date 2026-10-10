import { HttpResponse } from 'msw'
import { expect, test } from 'vitest'

import {
  handleSiteUserEmailChange,
  handleSiteUserGetMe,
  handleSiteUserResendVerification,
  handleSiteUserUpdateMe,
} from '../../generated/site/msw.gen.ts'
import type { SiteUserResource } from '../../generated/site/types.gen.ts'
import { problem } from '../../test/problem.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { worker } from '../../test/worker.ts'
import { ProfilePage } from './profile.tsx'

const user: SiteUserResource = {
  id: '1',
  name: 'John',
  email: 'info@1mail.com',
  emailVerified: false,
  createdAt: '2026-01-01T00:00:00Z',
}

const noContent = () => new Response(null, { status: 204 })

type Op = 'get' | 'update' | 'emailChange' | 'resend'
type Call = { op: Op; body: unknown }
type Handler = (req: Request) => Response | Promise<Response>

// Serves the profile endpoints and records every mutating call.
function serveProfile(
  calls: Call[],
  overrides: Partial<Record<Op, Handler>> = {},
  me: SiteUserResource = user,
) {
  const serve =
    (op: Op, fallback: Handler): Handler =>
    async (req) => {
      if (req.method !== 'GET') {
        const text = await req.clone().text()
        calls.push({ op, body: text ? JSON.parse(text) : null })
      }
      return (overrides[op] ?? fallback)(req)
    }
  const json = () => HttpResponse.json(me)
  worker.use(
    handleSiteUserGetMe(({ request }) => serve('get', json)(request)),
    handleSiteUserUpdateMe(({ request }) => serve('update', json)(request)),
    handleSiteUserEmailChange(({ request }) => serve('emailChange', noContent)(request)),
    handleSiteUserResendVerification(({ request }) => serve('resend', noContent)(request)),
  )
}

test('an unverified user can ask for the verification email again', async () => {
  const calls: Call[] = []
  serveProfile(calls)
  const { screen } = await renderWithRouter(<ProfilePage />)

  await expect.element(screen.getByText('Not verified')).toBeInTheDocument()
  await screen.getByRole('button', { name: 'Resend verification email' }).click()

  await expect.element(screen.getByText('Verification email sent')).toBeInTheDocument()
  expect(calls.map((c) => c.op)).toEqual(['resend'])
})

test('a verified user sees the verified badge and no resend prompt', async () => {
  serveProfile([], {}, { ...user, emailVerified: true })
  const { screen } = await renderWithRouter(<ProfilePage />)

  await expect.element(screen.getByText('Verified', { exact: true })).toBeInTheDocument()
  expect(screen.getByText('Your email address is not verified.').elements()).toHaveLength(0)
})

test('reports a failed verification resend', async () => {
  serveProfile([], {
    resend: () => problem(429, { detail: 'slow down' }),
  })
  const { screen } = await renderWithRouter(<ProfilePage />)

  await screen.getByRole('button', { name: 'Resend verification email' }).click()

  await expect.element(screen.getByText('Could not send verification email')).toBeInTheDocument()
})

test('changing the password sends the current and new password', async () => {
  const calls: Call[] = []
  serveProfile(calls)
  const { screen } = await renderWithRouter(<ProfilePage />)

  await expect.element(screen.getByLabelText(/^Name/)).toHaveValue('John')
  await screen.getByLabelText('Current password', { exact: true }).first().fill('old-pass')
  await screen.getByLabelText(/^New password/).fill('new-pass-123')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.element(screen.getByText('Profile updated')).toBeInTheDocument()
  expect(calls.find((c) => c.op === 'update')?.body).toEqual({
    name: 'John',
    currentPassword: 'old-pass',
    newPassword: 'new-pass-123',
  })
})

test('reports a failed profile update', async () => {
  serveProfile([], {
    update: () => problem(400, { detail: 'wrong password' }),
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
  expect(calls.find((c) => c.op === 'emailChange')?.body).toEqual({
    newEmail: 'next@1mail.com',
    currentPassword: 'pw-123456',
  })
  await expect.element(screen.getByLabelText(/^New email/)).toHaveValue('')
})

test('reports a failed email change', async () => {
  serveProfile([], {
    emailChange: () => problem(409, { detail: 'taken' }),
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
  worker.use(handleSiteUserGetMe(() => problem(500, { detail: 'boom' })))
  const { screen } = await renderWithRouter(<ProfilePage />)

  await expect.element(screen.getByText('Failed to load profile').first()).toBeInTheDocument()
})
