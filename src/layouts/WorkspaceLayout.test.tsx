import { useQuery } from '@tanstack/react-query'
import { expect, test } from 'vitest'

import { siteTagsListOptions } from '../generated/site/@tanstack/react-query.gen.ts'
import type {
  SiteTagsListData,
  SiteWorkspaceResource,
  SiteWorkspacesListData,
} from '../generated/site/types.gen.ts'
import { overviewRoute, securityRoute, workspaceRoute } from '../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../test/mockFetch.ts'
import { renderWithRouter } from '../test/renderWithRouter.tsx'
import { routeMount } from '../test/routeMount.ts'
import { WorkspaceLayout } from './WorkspaceLayout.tsx'

// route() wants a `path` record; these operations have none, so give it an empty one.
const MOUNT = routeMount(workspaceRoute, { slug: 'acme' })

const workspace = (over: Partial<SiteWorkspaceResource>): SiteWorkspaceResource => ({
  id: '1',
  name: 'Acme',
  slug: 'acme',
  collectKey: 'ck',
  ingestKey: 'ik',
  postalAddress: '',
  createdAt: '2026-01-01T00:00:00Z',
  ...over,
})

const list = (items: SiteWorkspaceResource[]) =>
  route<SiteWorkspacesListData>('GET', '/workspaces', {}, () => jsonResponse(items))

test('renders the workspace sidebar and no suspension banner for an active workspace', async () => {
  mockClientRoutes([list([workspace({})])])
  const { screen } = await renderWithRouter(<WorkspaceLayout />, MOUNT)

  await expect.element(screen.getByText('Overview', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('Sending is suspended')).not.toBeInTheDocument()
})

test('a suspended workspace shows the banner with the reason', async () => {
  mockClientRoutes([
    list([workspace({ suspendedAt: '2026-02-01T00:00:00Z', suspensionReason: 'spam' })]),
  ])
  const { screen } = await renderWithRouter(<WorkspaceLayout />, MOUNT)

  await expect.element(screen.getByText('Sending is suspended')).toBeInTheDocument()
  await expect.element(screen.getByText(/Reason: spam/)).toBeInTheDocument()
})

test('a suspension without a reason omits the reason line', async () => {
  mockClientRoutes([list([workspace({ suspendedAt: '2026-02-01T00:00:00Z' })])])
  const { screen } = await renderWithRouter(<WorkspaceLayout />, MOUNT)

  await expect.element(screen.getByText('Sending is suspended')).toBeInTheDocument()
  await expect.element(screen.getByText(/Reason:/)).not.toBeInTheDocument()
})

test('the switcher navigates to the chosen workspace overview', async () => {
  mockClientRoutes([list([workspace({}), workspace({ id: '2', name: 'Beta', slug: 'beta' })])])
  const { screen, navigate } = await renderWithRouter(<WorkspaceLayout />, MOUNT)

  await screen.getByRole('combobox').click()
  await screen.getByRole('option', { name: 'Beta' }).click()

  expect(navigate).toHaveBeenCalledWith({ to: overviewRoute.to, params: { slug: 'beta' } })
})

// TagsProbe stands in for a workspace page whose query the server refuses.
function TagsProbe() {
  useQuery({ ...siteTagsListOptions({ path: { slug: 'acme' } }), retry: false })
  return null
}

const DAY = 24 * 60 * 60 * 1000

test('during the grace of a Two-factor requirement a banner leads to enrollment', async () => {
  const endsAt = new Date(Date.now() + 3 * DAY).toISOString()
  mockClientRoutes([list([workspace({ secondFactorGraceEndsAt: endsAt })])])
  const { screen } = await renderWithRouter(<WorkspaceLayout />, MOUNT)

  await expect
    .element(screen.getByText('This workspace requires two-factor authentication'))
    .toBeInTheDocument()
  await expect
    .element(screen.getByRole('link', { name: 'Set up two-factor authentication' }))
    .toHaveAttribute('href', securityRoute.to)
  await expect
    .element(screen.getByText('Two-factor authentication required'))
    .not.toBeInTheDocument()
})

test('after the grace the workspace is replaced by the screen that leads to enrollment', async () => {
  const endsAt = new Date(Date.now() - DAY).toISOString()
  mockClientRoutes([list([workspace({ secondFactorGraceEndsAt: endsAt })])])
  const { screen } = await renderWithRouter(<WorkspaceLayout />, MOUNT)

  await expect.element(screen.getByText('Two-factor authentication required')).toBeInTheDocument()
  await expect
    .element(screen.getByRole('link', { name: 'Set up two-factor authentication' }))
    .toHaveAttribute('href', securityRoute.to)
})

test('a 403 second_factor_required from the workspace shows the blocked screen', async () => {
  mockClientRoutes([
    list([workspace({})]),
    route<SiteTagsListData>('GET', '/workspaces/{slug}/tags', { slug: 'acme' }, () =>
      jsonResponse(
        { status: 403, title: 'Forbidden', code: 'second_factor_required' },
        { status: 403 },
      ),
    ),
  ])
  const { screen } = await renderWithRouter(
    <>
      <WorkspaceLayout />
      <TagsProbe />
    </>,
    MOUNT,
  )

  await expect.element(screen.getByText('Two-factor authentication required')).toBeInTheDocument()
})
