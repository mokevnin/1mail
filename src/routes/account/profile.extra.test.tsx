import { expect, test } from 'vitest'

import type {
  SiteUserEmailChangeData,
  SiteUserGetMeData,
  SiteUserResendVerificationData,
  SiteUserResource,
  SiteUserSignOutEverywhereData,
  SiteUserUpdateMeData,
} from '../../generated/site/types.gen.ts'
import { loginRoute } from '../../router.tsx'
import { jsonResponse, mockClientFetch, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { ProfilePage } from './profile.tsx'

const user: SiteUserResource = {
  id: '1',
  name: 'John',
  email: 'info@1mail.com',
  emailVerified: false,
  createdAt: '2026-01-01T00:00:00Z',
}

const noContent = () => new Response(null, { status: 204 })

type Op = 'get' | 'update' | 'emailChange' | 'resend' | 'signOutEverywhere'
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
  mockClientRoutes([
    route<SiteUserGetMeData>(
      'GET',
      '/me',
      {},
      serve('get', () => jsonResponse(me)),
    ),
    route<SiteUserUpdateMeData>(
      'PUT',
      '/me',
      {},
      serve('update', () => jsonResponse(me)),
    ),
    route<SiteUserEmailChangeData>('POST', '/me/email-change', {}, serve('emailChange', noContent)),
    route<SiteUserResendVerificationData>(
      'POST',
      '/me/verification-email',
      {},
      serve('resend', noContent),
    ),
    route<SiteUserSignOutEverywhereData>(
      'POST',
      '/me/sign-out-everywhere',
      {},
      serve('signOutEverywhere', noContent),
    ),
  ])
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
    resend: () => jsonResponse({ status: 429, detail: 'slow down' }, { status: 429 }),
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
    update: () => jsonResponse({ status: 400, detail: 'wrong password' }, { status: 400 }),
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
    emailChange: () => jsonResponse({ status: 409, detail: 'taken' }, { status: 409 }),
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

test('signing out everywhere asks first, then ends the sessions and goes to login', async () => {
  const calls: Call[] = []
  serveProfile(calls)
  const { screen, navigate } = await renderWithRouter(<ProfilePage />)

  await screen.getByRole('button', { name: 'Sign out everywhere' }).click()
  expect(calls).toEqual([])
  await screen.getByRole('dialog').getByRole('button', { name: 'Sign out everywhere' }).click()

  await expect.poll(() => calls.map((c) => c.op)).toEqual(['signOutEverywhere'])
  await expect.poll(() => navigate).toHaveBeenCalledWith({ to: loginRoute.to })
})

test('reports a failed sign out everywhere and stays put', async () => {
  serveProfile([], {
    signOutEverywhere: () => jsonResponse({ status: 500, detail: 'boom' }, { status: 500 }),
  })
  const { screen, navigate } = await renderWithRouter(<ProfilePage />)

  await screen.getByRole('button', { name: 'Sign out everywhere' }).click()
  await screen.getByRole('dialog').getByRole('button', { name: 'Sign out everywhere' }).click()

  await expect.element(screen.getByText('Could not sign out everywhere')).toBeInTheDocument()
  expect(navigate).not.toHaveBeenCalledWith({ to: loginRoute.to })
})
