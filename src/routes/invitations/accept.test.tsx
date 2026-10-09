import { expect, test } from 'vitest'

import type {
  SiteInvitationLookupResult,
  SitePublicInvitationsAcceptData,
  SitePublicInvitationsLookupData,
} from '../../generated/site/types.gen.ts'
import { acceptInvitationRoute, loginRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { AcceptInvitationPage } from './accept.tsx'

const MOUNT = routeMount(acceptInvitationRoute, { token: 'inv-1' })
const TOKEN = { token: 'inv-1' }

const lookup = (respond: () => Response | Promise<Response>) =>
  route<SitePublicInvitationsLookupData>('GET', '/invitations/{token}', TOKEN, respond)
const accept = (respond: (req: Request) => Response | Promise<Response>) =>
  route<SitePublicInvitationsAcceptData>('POST', '/invitations/{token}/accept', TOKEN, respond)

const invite = (hasAccount: boolean): SiteInvitationLookupResult => ({
  workspaceName: 'Acme',
  email: 'ada@example.com',
  hasAccount,
})

test('a new user supplies a name and password to accept', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    lookup(() => jsonResponse(invite(false))),
    accept(async (req) => {
      bodies.push(await req.json())
      return jsonResponse({})
    }),
  ])
  const { screen, navigate } = await renderWithRouter(<AcceptInvitationPage />, MOUNT)

  await expect
    .element(screen.getByText("You've been invited to join Acme on 1mail."))
    .toBeInTheDocument()
  await expect.element(screen.getByText('ada@example.com')).toBeInTheDocument()
  await screen.getByLabelText(/^Your name/).fill('  Ada  ')
  await screen.getByLabelText(/^Password/).fill('s3cret')
  await screen.getByRole('button', { name: 'Accept invitation' }).click()

  await expect.element(screen.getByText('Invitation accepted')).toBeInTheDocument()
  expect(bodies).toEqual([{ name: 'Ada', password: 's3cret' }])
  await expect.poll(() => navigate.mock.calls).toContainEqual([{ to: loginRoute.to }])
})

test('an existing account joins with an empty body and no credential fields', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    lookup(() => jsonResponse(invite(true))),
    accept(async (req) => {
      bodies.push(await req.json())
      return jsonResponse({})
    }),
  ])
  const { screen } = await renderWithRouter(<AcceptInvitationPage />, MOUNT)

  await expect.element(screen.getByLabelText(/^Password/)).not.toBeInTheDocument()
  await screen.getByRole('button', { name: 'Join workspace' }).click()

  await expect.element(screen.getByText('Invitation accepted')).toBeInTheDocument()
  expect(bodies).toEqual([{}])
})

test('an accept failure shows the API error and stays on the page', async () => {
  mockClientRoutes([
    lookup(() => jsonResponse(invite(true))),
    accept(() => jsonResponse({ detail: 'Invitation revoked' }, { status: 410 })),
  ])
  const { screen, navigate } = await renderWithRouter(<AcceptInvitationPage />, MOUNT)

  await screen.getByRole('button', { name: 'Join workspace' }).click()

  await expect.element(screen.getByText('Invitation revoked')).toBeInTheDocument()
  expect(navigate).not.toHaveBeenCalled()
})

test('shows a loading state while the invitation is looked up', async () => {
  mockClientRoutes([lookup(() => new Promise<Response>(() => undefined))])
  const { screen } = await renderWithRouter(<AcceptInvitationPage />, MOUNT)

  await expect.element(screen.getByText('Loading invitation…')).toBeInTheDocument()
})

test('an unknown invitation shows the invalid alert', async () => {
  mockClientRoutes([lookup(() => jsonResponse({ status: 404 }, { status: 404 }))])
  const { screen } = await renderWithRouter(<AcceptInvitationPage />, MOUNT)

  await expect
    .element(screen.getByText('This invitation is invalid or has expired.'))
    .toBeInTheDocument()
})
