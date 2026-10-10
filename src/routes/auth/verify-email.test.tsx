import { HttpResponse } from 'msw'
import { expect, test } from 'vitest'

import { handleSiteAuthVerifyEmail } from '../../generated/site/msw.gen.ts'
import { indexRoute, verifyEmailRoute } from '../../router.tsx'
import { problem } from '../../test/problem.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { worker } from '../../test/worker.ts'
import { VerifyEmailPage } from './verify-email.tsx'

const mount = routeMount(verifyEmailRoute)
const MOUNT = { path: mount.path, initialPath: `${mount.initialPath}?token=tok-v` }

test('verifies the token on mount and offers to continue', async () => {
  const bodies: unknown[] = []
  worker.use(
    handleSiteAuthVerifyEmail(async ({ request }) => {
      bodies.push(await request.json())
      return HttpResponse.json({})
    }),
  )
  const { screen } = await renderWithRouter(<VerifyEmailPage />, MOUNT)

  await expect.element(screen.getByText('Email verified')).toBeInTheDocument()
  expect(bodies).toEqual([{ token: 'tok-v' }])
  await expect
    .element(screen.getByRole('link', { name: 'Continue' }))
    .toHaveAttribute('href', indexRoute.to)
})

test('shows a loading state while the request is in flight', async () => {
  worker.use(handleSiteAuthVerifyEmail(() => new Promise<never>(() => undefined)))
  const { screen } = await renderWithRouter(<VerifyEmailPage />, MOUNT)

  await expect.element(screen.getByText('Verifying your email…')).toBeInTheDocument()
})

test('shows the failure state when the token is rejected', async () => {
  worker.use(handleSiteAuthVerifyEmail(() => problem(400, {})))
  const { screen } = await renderWithRouter(<VerifyEmailPage />, MOUNT)

  await expect.element(screen.getByText('Verification failed')).toBeInTheDocument()
  await expect
    .element(screen.getByText('This verification link is invalid or has expired.'))
    .toBeInTheDocument()
})
