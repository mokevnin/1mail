import { useQuery } from '@tanstack/react-query'
import { expect, test } from 'vitest'

import {
  siteSegmentsGetOptions,
  siteSegmentsListOptions,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type {
  SiteCustomFieldsListData,
  SiteEventsActionsData,
  SiteSegmentsCreateData,
  SiteSegmentsGetData,
  SiteSegmentsListData,
  SiteSegmentsUpdateData,
} from '../../generated/site/types.gen.ts'
import { segmentsCreateRoute, segmentsEditRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { SegmentCreatePage, SegmentEditPage } from './resource.tsx'

const RULE = '{"combinator":"and","rules":[{"field":"email","operator":"contains","value":"@x"}]}'

const CREATE_ROUTE = routeMount(segmentsCreateRoute, { slug: 'test' })

const EDIT_ROUTE = routeMount(segmentsEditRoute, { slug: 'test', segmentId: '7' })

// Stands in for the segments list screen: an active list query, so invalidation after a save
// is observable as a refetch.
function ListProbe() {
  useQuery(siteSegmentsListOptions({ path: { slug: 'test' } }))
  return null
}

function DetailProbe() {
  useQuery(siteSegmentsGetOptions({ path: { slug: 'test', id: '7' } }))
  return null
}

const SLUG = { slug: 'test' }
const ID7 = { slug: 'test', id: '7' }

// Everything the segment form reads besides the segment itself (rule builder catalogues).
const CATALOGUE_ROUTES = [
  route<SiteEventsActionsData>('GET', '/workspaces/{slug}/events/actions', SLUG, () =>
    jsonResponse({ actions: [] }),
  ),
  route<SiteCustomFieldsListData>('GET', '/workspaces/{slug}/custom-fields', SLUG, () =>
    jsonResponse({ items: [], totalItems: 0 }),
  ),
]

test('creating a segment sends name and a rule, refreshes the list and opens the edit page; there is no type selector', async () => {
  const bodies: unknown[] = []
  let listFetches = 0
  mockClientRoutes([
    ...CATALOGUE_ROUTES,
    route<SiteSegmentsCreateData>('POST', '/workspaces/{slug}/segments', SLUG, async (req) => {
      bodies.push(await req.json())
      return jsonResponse({ id: '42', name: 'Everyone' }, { status: 201 })
    }),
    route<SiteSegmentsListData>('GET', '/workspaces/{slug}/segments', SLUG, () => {
      listFetches++
      return jsonResponse({ items: [], totalItems: 0 })
    }),
  ])

  const { screen, navigate } = await renderWithRouter(
    <>
      <ListProbe />
      <SegmentCreatePage />
    </>,
    CREATE_ROUTE,
  )
  await expect.poll(() => listFetches).toBe(1)
  expect(screen.getByLabelText('Type').elements()).toEqual([])

  await screen.getByLabelText('Name', { exact: false }).fill('Everyone')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.poll(() => bodies.length).toBe(1)
  // The rule builder stamps its own id on the group; the rule itself is the empty group.
  expect(bodies[0]).toEqual({ name: 'Everyone', definition: expect.any(String) })
  expect(JSON.parse(String(Reflect.get(bodies[0] ?? {}, 'definition')))).toMatchObject({
    combinator: 'and',
    rules: [],
  })
  await expect.element(screen.getByText('Segment created')).toBeInTheDocument()
  await expect.poll(() => listFetches).toBe(2)
  await expect.poll(() => navigate.mock.calls.length).toBe(1)
  expect(navigate.mock.calls[0]?.[0]).toMatchObject({
    params: { slug: 'test', segmentId: '42' },
  })
})

test('a blank name is rejected before anything is sent', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    ...CATALOGUE_ROUTES,
    route<SiteSegmentsCreateData>('POST', '/workspaces/{slug}/segments', SLUG, async (req) => {
      bodies.push(await req.json())
      return jsonResponse({ id: '1' }, { status: 201 })
    }),
  ])

  const { screen } = await renderWithRouter(<SegmentCreatePage />, CREATE_ROUTE)

  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.element(screen.getByLabelText('Name', { exact: false })).toBeInvalid()
  expect(bodies).toEqual([])
})

test('editing shows the stored rule untouched, saves a renamed segment and refreshes list and detail', async () => {
  const bodies: unknown[] = []
  let detailFetches = 0
  let listFetches = 0
  mockClientRoutes([
    ...CATALOGUE_ROUTES,
    route<SiteSegmentsUpdateData>('PUT', '/workspaces/{slug}/segments/{id}', ID7, async (req) => {
      bodies.push(await req.json())
      return jsonResponse({ id: '7', name: 'Renamed', definition: RULE })
    }),
    route<SiteSegmentsListData>('GET', '/workspaces/{slug}/segments', SLUG, () => {
      listFetches++
      return jsonResponse({ items: [], totalItems: 0 })
    }),
    route<SiteSegmentsGetData>('GET', '/workspaces/{slug}/segments/{id}', ID7, () => {
      detailFetches++
      return jsonResponse({ id: '7', name: 'Mail users', definition: RULE })
    }),
  ])

  const { screen } = await renderWithRouter(
    <>
      <ListProbe />
      <DetailProbe />
      <SegmentEditPage />
    </>,
    EDIT_ROUTE,
  )

  await expect.element(screen.getByLabelText('Name', { exact: false })).toHaveValue('Mail users')
  expect(screen.getByLabelText('Type').elements()).toEqual([])

  await screen.getByLabelText('Name', { exact: false }).fill('Renamed')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.poll(() => bodies).toEqual([{ name: 'Renamed', definition: RULE }])
  await expect.element(screen.getByText('Segment updated')).toBeInTheDocument()
  await expect.poll(() => listFetches).toBe(2)
  await expect.poll(() => detailFetches).toBeGreaterThan(1)
})

test('editing shows an error alert when the segment cannot be loaded', async () => {
  mockClientRoutes([
    ...CATALOGUE_ROUTES,
    route<SiteSegmentsGetData>('GET', '/workspaces/{slug}/segments/{id}', ID7, () =>
      jsonResponse({ title: 'Not Found', detail: 'segment not found' }, { status: 404 }),
    ),
  ])

  const { screen } = await renderWithRouter(<SegmentEditPage />, EDIT_ROUTE)

  await expect.element(screen.getByText('Failed to load segments')).toBeVisible()
  await expect.element(screen.getByText('segment not found')).toBeVisible()
})

test('editing shows no form while the segment loads, then the loaded values', async () => {
  const gate = Promise.withResolvers<void>()
  mockClientRoutes([
    ...CATALOGUE_ROUTES,
    route<SiteSegmentsGetData>('GET', '/workspaces/{slug}/segments/{id}', ID7, async () => {
      await gate.promise
      return jsonResponse({ id: '7', name: 'Mail users', definition: RULE })
    }),
  ])

  const { screen } = await renderWithRouter(<SegmentEditPage />, EDIT_ROUTE)

  await expect.element(screen.getByLabelText('Name', { exact: false })).not.toBeInTheDocument()
  gate.resolve()

  await expect.element(screen.getByLabelText('Name', { exact: false })).toHaveValue('Mail users')
})
