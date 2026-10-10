import { expect, test } from 'vitest'

import { handleSiteWorkspacesList } from '../generated/site/msw.gen.ts'
import type { SiteWorkspaceResource } from '../generated/site/types.gen.ts'
import { overviewRoute, workspaceRoute } from '../router.tsx'
import { renderWithRouter } from '../test/renderWithRouter.tsx'
import { routeMount } from '../test/routeMount.ts'
import { worker } from '../test/worker.ts'
import { WorkspaceLayout } from './WorkspaceLayout.tsx'

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

const list = (items: SiteWorkspaceResource[]) => handleSiteWorkspacesList({ body: items })

test('renders the workspace sidebar and no suspension banner for an active workspace', async () => {
  worker.use(list([workspace({})]))
  const { screen } = await renderWithRouter(<WorkspaceLayout />, MOUNT)

  await expect.element(screen.getByText('Overview', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('Sending is suspended')).not.toBeInTheDocument()
})

test('a suspended workspace shows the banner with the reason', async () => {
  worker.use(list([workspace({ suspendedAt: '2026-02-01T00:00:00Z', suspensionReason: 'spam' })]))
  const { screen } = await renderWithRouter(<WorkspaceLayout />, MOUNT)

  await expect.element(screen.getByText('Sending is suspended')).toBeInTheDocument()
  await expect.element(screen.getByText(/Reason: spam/)).toBeInTheDocument()
})

test('a suspension without a reason omits the reason line', async () => {
  worker.use(list([workspace({ suspendedAt: '2026-02-01T00:00:00Z' })]))
  const { screen } = await renderWithRouter(<WorkspaceLayout />, MOUNT)

  await expect.element(screen.getByText('Sending is suspended')).toBeInTheDocument()
  await expect.element(screen.getByText(/Reason:/)).not.toBeInTheDocument()
})

test('the switcher navigates to the chosen workspace overview', async () => {
  worker.use(list([workspace({}), workspace({ id: '2', name: 'Beta', slug: 'beta' })]))
  const { screen, navigate } = await renderWithRouter(<WorkspaceLayout />, MOUNT)

  await screen.getByRole('combobox').click()
  await screen.getByRole('option', { name: 'Beta' }).click()

  expect(navigate).toHaveBeenCalledWith({ to: overviewRoute.to, params: { slug: 'beta' } })
})
