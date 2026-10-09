import { expect, test } from 'vitest'

import type {
  SiteSegmentsDeleteData,
  SiteSegmentsListData,
} from '../../generated/site/types.gen.ts'
import { segmentsRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { SegmentsListPage } from './list.tsx'

const SLUG = { slug: 'test' }
const LIST_ROUTE = routeMount(segmentsRoute, SLUG)

const VIPS = { id: '1', name: 'VIP customers' }
const TRIAL = { id: '2', name: 'Trial users' }

function listRoute(respond: () => Response) {
  return route<SiteSegmentsListData>('GET', '/workspaces/{slug}/segments', SLUG, respond)
}

test('lists the workspace segments', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [VIPS, TRIAL], totalItems: 2 }))])

  const { screen } = await renderWithRouter(<SegmentsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('VIP customers')).toBeInTheDocument()
  await expect.element(screen.getByText('Trial users')).toBeInTheDocument()
})

test('shows the empty state when there are no segments', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [], totalItems: 0 }))])

  const { screen } = await renderWithRouter(<SegmentsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Create your first segment.')).toBeInTheDocument()
})

test('shows an error alert when the list fails to load', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ title: 'Boom', status: 500 }, { status: 500 }))])

  const { screen } = await renderWithRouter(<SegmentsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Failed to load segments')).toBeInTheDocument()
})

test('Add segment navigates to the create page', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [], totalItems: 0 }))])

  const { screen, navigate } = await renderWithRouter(<SegmentsListPage />, LIST_ROUTE)
  await screen.getByRole('button', { name: 'Add segment' }).click()

  expect(navigate).toHaveBeenCalledWith(expect.objectContaining({ params: SLUG }))
})

test('Edit navigates to the segment edit page', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [VIPS], totalItems: 1 }))])

  const { screen, navigate } = await renderWithRouter(<SegmentsListPage />, LIST_ROUTE)
  await expect.element(screen.getByText('VIP customers')).toBeInTheDocument()
  await screen.getByRole('button', { name: 'Edit' }).click()

  expect(navigate).toHaveBeenCalledWith(
    expect.objectContaining({ params: { slug: 'test', segmentId: '1' } }),
  )
})

test('deleting a segment asks for confirmation, deletes it and refreshes the list', async () => {
  const deleted: string[] = []
  let listFetches = 0
  mockClientRoutes([
    listRoute(() => {
      listFetches++
      return jsonResponse({ items: deleted.length ? [TRIAL] : [VIPS, TRIAL], totalItems: 2 })
    }),
    route<SiteSegmentsDeleteData>(
      'DELETE',
      '/workspaces/{slug}/segments/{id}',
      { ...SLUG, id: '1' },
      () => {
        deleted.push('1')
        return new Response(null, { status: 204 })
      },
    ),
  ])

  const { screen } = await renderWithRouter(<SegmentsListPage />, LIST_ROUTE)
  await expect.element(screen.getByText('VIP customers')).toBeInTheDocument()

  await screen.getByRole('button', { name: 'Delete' }).first().click()
  expect(deleted).toEqual([])
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => deleted).toEqual(['1'])
  await expect.element(screen.getByText('Segment deleted')).toBeInTheDocument()
  await expect.poll(() => listFetches).toBe(2)
  await expect.element(screen.getByText('VIP customers')).not.toBeInTheDocument()
})
