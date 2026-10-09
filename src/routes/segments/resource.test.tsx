import { useQuery } from '@tanstack/react-query'
import { expect, test } from 'vitest'

import {
  siteSegmentsGetOptions,
  siteSegmentsListOptions,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { SegmentCreatePage, SegmentEditPage } from './resource.tsx'

const RULE = '{"combinator":"and","rules":[{"field":"email","operator":"contains","value":"@x"}]}'

const CREATE_ROUTE = {
  path: '/workspaces/$slug/segments/new',
  initialPath: '/workspaces/test/segments/new',
}

const EDIT_ROUTE = {
  path: '/workspaces/$slug/segments/$segmentId/edit',
  initialPath: '/workspaces/test/segments/7/edit',
}

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

function requestOf(input: RequestInfo | URL, init?: RequestInit) {
  return input instanceof Request ? input : new Request(String(input), init)
}

function pathOf(req: Request) {
  return new URL(req.url, 'http://x').pathname
}

// Everything the segment form reads besides the segment itself (rule builder catalogues).
function sideResponse(req: Request) {
  const path = pathOf(req)
  if (path.endsWith('/events/actions')) return jsonResponse({ actions: [] })
  if (path.endsWith('/custom-fields')) return jsonResponse({ items: [], totalItems: 0 })
  return undefined
}

test('creating a segment sends name and a rule, refreshes the list and opens the edit page; there is no type selector', async () => {
  const bodies: unknown[] = []
  let listFetches = 0
  mockClientFetch(async (input, init) => {
    const req = requestOf(input, init)
    if (req.method === 'POST') {
      bodies.push(await req.json())
      return jsonResponse({ id: '42', name: 'Everyone' }, { status: 201 })
    }
    const side = sideResponse(req)
    if (side) return side
    listFetches++
    return jsonResponse({ items: [], totalItems: 0 })
  })

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
  mockClientFetch(async (input, init) => {
    const req = requestOf(input, init)
    const side = sideResponse(req)
    if (side) return side
    bodies.push(await req.json())
    return jsonResponse({ id: '1' }, { status: 201 })
  })

  const { screen } = await renderWithRouter(<SegmentCreatePage />, CREATE_ROUTE)

  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.element(screen.getByLabelText('Name', { exact: false })).toBeInvalid()
  expect(bodies).toEqual([])
})

test('editing shows the stored rule untouched, saves a renamed segment and refreshes list and detail', async () => {
  const bodies: unknown[] = []
  let detailFetches = 0
  let listFetches = 0
  mockClientFetch(async (input, init) => {
    const req = requestOf(input, init)
    if (req.method === 'PUT' || req.method === 'PATCH') {
      bodies.push(await req.json())
      return jsonResponse({ id: '7', name: 'Renamed', definition: RULE })
    }
    const side = sideResponse(req)
    if (side) return side
    if (pathOf(req).endsWith('/segments')) {
      listFetches++
      return jsonResponse({ items: [], totalItems: 0 })
    }
    detailFetches++
    return jsonResponse({ id: '7', name: 'Mail users', definition: RULE })
  })

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
  mockClientFetch((input, init) => {
    const side = sideResponse(requestOf(input, init))
    return (
      side ?? jsonResponse({ title: 'Not Found', detail: 'segment not found' }, { status: 404 })
    )
  })

  const { screen } = await renderWithRouter(<SegmentEditPage />, EDIT_ROUTE)

  await expect.element(screen.getByText('Failed to load segments')).toBeVisible()
  await expect.element(screen.getByText('segment not found')).toBeVisible()
})
