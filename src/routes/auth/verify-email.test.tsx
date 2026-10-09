import { expect, test } from 'vitest'

import type { SiteAuthVerifyEmailData } from '../../generated/site/types.gen.ts'
import { indexRoute, verifyEmailRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { VerifyEmailPage } from './verify-email.tsx'

// route() wants a `path` record; these operations have none, so give it an empty one.
const mount = routeMount(verifyEmailRoute)
const MOUNT = { path: mount.path, initialPath: `${mount.initialPath}?token=tok-v` }

const verify = (respond: (req: Request) => Response | Promise<Response>) =>
  route<SiteAuthVerifyEmailData>('POST', '/auth/verify-email', {}, respond)

test('verifies the token on mount and offers to continue', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    verify(async (req) => {
      bodies.push(await req.json())
      return jsonResponse({})
    }),
  ])
  const { screen } = await renderWithRouter(<VerifyEmailPage />, MOUNT)

  await expect.element(screen.getByText('Email verified')).toBeInTheDocument()
  expect(bodies).toEqual([{ token: 'tok-v' }])
  await expect
    .element(screen.getByRole('link', { name: 'Continue' }))
    .toHaveAttribute('href', indexRoute.to)
})

test('shows a loading state while the request is in flight', async () => {
  mockClientRoutes([verify(() => new Promise<Response>(() => undefined))])
  const { screen } = await renderWithRouter(<VerifyEmailPage />, MOUNT)

  await expect.element(screen.getByText('Verifying your email…')).toBeInTheDocument()
})

test('shows the failure state when the token is rejected', async () => {
  mockClientRoutes([verify(() => jsonResponse({ status: 400 }, { status: 400 }))])
  const { screen } = await renderWithRouter(<VerifyEmailPage />, MOUNT)

  await expect.element(screen.getByText('Verification failed')).toBeInTheDocument()
  await expect
    .element(screen.getByText('This verification link is invalid or has expired.'))
    .toBeInTheDocument()
})
