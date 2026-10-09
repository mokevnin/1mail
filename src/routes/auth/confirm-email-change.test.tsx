import { expect, test } from 'vitest'

import type { SiteAuthConfirmEmailChangeData } from '../../generated/site/types.gen.ts'
import { confirmEmailChangeRoute, loginRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { ConfirmEmailChangePage } from './confirm-email-change.tsx'

// route() wants a `path` record; these operations have none, so give it an empty one.
type PathlessOperation<D extends { url: string }> = { url: D['url']; path: Record<string, string> }

const mount = routeMount(confirmEmailChangeRoute)
const MOUNT = { path: mount.path, initialPath: `${mount.initialPath}?token=tok-c` }

const confirm = (respond: (req: Request) => Response | Promise<Response>) =>
  route<PathlessOperation<SiteAuthConfirmEmailChangeData>>(
    'POST',
    '/auth/confirm-email-change',
    {},
    respond,
  )

test('confirms the new email on mount and links to sign in', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    confirm(async (req) => {
      bodies.push(await req.json())
      return jsonResponse({})
    }),
  ])
  const { screen } = await renderWithRouter(<ConfirmEmailChangePage />, MOUNT)

  await expect.element(screen.getByText('Email updated')).toBeInTheDocument()
  expect(bodies).toEqual([{ token: 'tok-c' }])
  await expect
    .element(screen.getByRole('link', { name: 'Go to sign in' }))
    .toHaveAttribute('href', loginRoute.to)
})

test('shows a loading state while the request is in flight', async () => {
  mockClientRoutes([confirm(() => new Promise<Response>(() => undefined))])
  const { screen } = await renderWithRouter(<ConfirmEmailChangePage />, MOUNT)

  await expect.element(screen.getByText('Confirming your new email…')).toBeInTheDocument()
})

test('shows the failure state when the token is rejected', async () => {
  mockClientRoutes([confirm(() => jsonResponse({ status: 400 }, { status: 400 }))])
  const { screen } = await renderWithRouter(<ConfirmEmailChangePage />, MOUNT)

  await expect.element(screen.getByText('Confirmation failed')).toBeInTheDocument()
  await expect
    .element(screen.getByText('This confirmation link is invalid or has expired.'))
    .toBeInTheDocument()
})
