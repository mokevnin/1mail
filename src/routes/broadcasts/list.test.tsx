import { expect, test } from 'vitest'

import type {
  SiteBroadcastsDeleteData,
  SiteBroadcastsListData,
} from '../../generated/site/types.gen.ts'
import { broadcastsRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { BroadcastsListPage } from './list.tsx'

const SLUG = { slug: 'test' }
const LIST_ROUTE = routeMount(broadcastsRoute, SLUG)

const STATS = { recipientsTotal: 120, openedCount: 45 }
const DRAFT = { id: '1', name: 'Spring launch', status: 'draft', stats: STATS }
const HELD = {
  id: '2',
  name: 'Autumn digest',
  status: 'sending',
  holdReason: 'unverified_domain',
  stats: STATS,
}

function listRoute(respond: () => Response) {
  return route<SiteBroadcastsListData>('GET', '/workspaces/{slug}/broadcasts', SLUG, respond)
}

test('lists broadcasts with status, hold badge and counters', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [DRAFT, HELD], totalItems: 2 }))])

  const { screen } = await renderWithRouter(<BroadcastsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Spring launch')).toBeInTheDocument()
  await expect.element(screen.getByText('Autumn digest')).toBeInTheDocument()
  await expect.element(screen.getByText('Draft')).toBeInTheDocument()
  await expect.element(screen.getByText('Sending', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('On hold')).toBeInTheDocument()
  await expect.element(screen.getByText('45').first()).toBeInTheDocument()
})

test('shows the empty state when there are no broadcasts', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [], totalItems: 0 }))])

  const { screen } = await renderWithRouter(<BroadcastsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Create your first broadcast.')).toBeInTheDocument()
})

test('shows an error alert when the list fails to load', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ title: 'Boom', status: 500 }, { status: 500 }))])

  const { screen } = await renderWithRouter(<BroadcastsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Failed to load broadcasts')).toBeInTheDocument()
})

test('New broadcast navigates to the create page', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [], totalItems: 0 }))])

  const { screen, navigate } = await renderWithRouter(<BroadcastsListPage />, LIST_ROUTE)
  await screen.getByRole('button', { name: 'New broadcast' }).click()

  expect(navigate).toHaveBeenCalledWith(expect.objectContaining({ params: SLUG }))
})

test('Report and Edit navigate with the broadcast id; Edit is only enabled for drafts', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [DRAFT, HELD], totalItems: 2 }))])

  const { screen, navigate } = await renderWithRouter(<BroadcastsListPage />, LIST_ROUTE)
  await expect.element(screen.getByText('Spring launch')).toBeInTheDocument()

  await expect.element(screen.getByRole('button', { name: 'Edit' }).nth(1)).toBeDisabled()

  await screen.getByRole('button', { name: 'Report' }).first().click()
  expect(navigate).toHaveBeenLastCalledWith(
    expect.objectContaining({ params: { slug: 'test', broadcastId: '1' } }),
  )

  await screen.getByRole('button', { name: 'Edit' }).first().click()
  expect(navigate).toHaveBeenLastCalledWith(
    expect.objectContaining({ params: { slug: 'test', broadcastId: '1' } }),
  )
  expect(navigate).toHaveBeenCalledTimes(2)
})

test('deleting a broadcast asks for confirmation, deletes it and refreshes the list', async () => {
  const deleted: string[] = []
  let listFetches = 0
  mockClientRoutes([
    listRoute(() => {
      listFetches++
      return jsonResponse({ items: deleted.length ? [HELD] : [DRAFT, HELD], totalItems: 2 })
    }),
    route<SiteBroadcastsDeleteData>(
      'DELETE',
      '/workspaces/{slug}/broadcasts/{id}',
      { ...SLUG, id: '1' },
      () => {
        deleted.push('1')
        return new Response(null, { status: 204 })
      },
    ),
  ])

  const { screen } = await renderWithRouter(<BroadcastsListPage />, LIST_ROUTE)
  await expect.element(screen.getByText('Spring launch')).toBeInTheDocument()

  await screen.getByRole('button', { name: 'Delete' }).first().click()
  expect(deleted).toEqual([])
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => deleted).toEqual(['1'])
  await expect.element(screen.getByText('Broadcast deleted')).toBeInTheDocument()
  await expect.poll(() => listFetches).toBe(2)
  await expect.element(screen.getByText('Spring launch')).not.toBeInTheDocument()
})
