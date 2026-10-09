import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { expect, test } from 'vitest'

import type {
  SiteContactResource,
  SiteContactsGetData,
  SiteEventsListData,
} from '../../generated/site/types.gen.ts'
import { contactsDetailRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithProviders } from '../../test/renderWithProviders.tsx'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { ContactDetailPage } from './detail.tsx'

const SLUG = { slug: 'test' }
const DETAIL_ROUTE = routeMount(contactsDetailRoute, { slug: 'test', contactId: '9' })
const NOW = '2026-01-01T00:00:00Z'

const ADA: SiteContactResource = {
  id: '9',
  email: 'ada@example.com',
  firstName: 'Ada',
  lastName: 'Lovelace',
  customFields: { plan: 'pro', seats: 3, flags: { beta: true }, nothing: null },
  createdAt: NOW,
  updatedAt: NOW,
}

function contactRoute(respond: () => Response) {
  return route<SiteContactsGetData>(
    'GET',
    '/workspaces/{slug}/contacts/{id}',
    { ...SLUG, id: '9' },
    respond,
  )
}

function eventsRoute(respond: () => Response) {
  return route<SiteEventsListData>('GET', '/workspaces/{slug}/events', SLUG, respond)
}

test('shows the contact, its custom fields and its events', async () => {
  mockClientRoutes([
    contactRoute(() => jsonResponse(ADA)),
    eventsRoute(() => jsonResponse({ items: [], totalItems: 0 })),
  ])

  const { screen } = await renderWithRouter(<ContactDetailPage />, DETAIL_ROUTE)

  await expect.element(screen.getByRole('heading', { name: 'ada@example.com' })).toBeInTheDocument()
  await expect.element(screen.getByText('Ada Lovelace')).toBeInTheDocument()
  await expect.element(screen.getByText('Custom fields')).toBeInTheDocument()
  await expect.element(screen.getByText('pro', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('{"beta":true}')).toBeInTheDocument()
  await expect.element(screen.getByText('No events yet')).toBeInTheDocument()
})

test('lists the contact events and filters them by contact id', async () => {
  let requestedContactId: string | null = null
  mockClientRoutes([
    contactRoute(() => jsonResponse({ ...ADA, customFields: null })),
    route<SiteEventsListData>('GET', '/workspaces/{slug}/events', SLUG, (req) => {
      requestedContactId = new URL(req.url).searchParams.get('contactId')
      return jsonResponse({
        items: [{ id: 'e1', subjectId: 's1', action: 'page.viewed', createdAt: NOW }],
        totalItems: 1,
      })
    }),
  ])

  const { screen } = await renderWithRouter(<ContactDetailPage />, DETAIL_ROUTE)

  await expect.element(screen.getByText('page.viewed')).toBeInTheDocument()
  expect(requestedContactId).toBe('9')
  await expect.element(screen.getByText('Custom fields')).not.toBeInTheDocument()
})

test('falls back to the subject id as the title when there is no email', async () => {
  mockClientRoutes([
    contactRoute(() =>
      jsonResponse({ id: '9', subjectId: 'anon-1', createdAt: NOW, updatedAt: NOW }),
    ),
    eventsRoute(() => jsonResponse({ items: [], totalItems: 0 })),
  ])

  const { screen } = await renderWithRouter(<ContactDetailPage />, DETAIL_ROUTE)

  await expect.element(screen.getByRole('heading', { name: 'anon-1' })).toBeInTheDocument()
})

test('shows an error alert when the contact fails to load', async () => {
  mockClientRoutes([
    contactRoute(() => jsonResponse({ title: 'Gone', status: 404 }, { status: 404 })),
    eventsRoute(() => jsonResponse({ items: [], totalItems: 0 })),
  ])

  const { screen } = await renderWithRouter(<ContactDetailPage />, DETAIL_ROUTE)

  await expect.element(screen.getByText('Failed to load contact').first()).toBeInTheDocument()
})

test('shows an activity error when the events fail to load', async () => {
  mockClientRoutes([
    contactRoute(() => jsonResponse(ADA)),
    eventsRoute(() => jsonResponse({ title: 'Boom', status: 500 }, { status: 500 })),
  ])

  const { screen } = await renderWithRouter(<ContactDetailPage />, DETAIL_ROUTE)

  await expect.element(screen.getByText('Failed to load activity').first()).toBeInTheDocument()
})

test('moving to another contact resets the events page to the first', async () => {
  const eventRequests: { contactId: string | null; page: string | null }[] = []
  mockClientRoutes([
    route<SiteContactsGetData>(
      'GET',
      '/workspaces/{slug}/contacts/{id}',
      { ...SLUG, id: '9' },
      () => jsonResponse(ADA),
    ),
    route<SiteContactsGetData>(
      'GET',
      '/workspaces/{slug}/contacts/{id}',
      { ...SLUG, id: '10' },
      () => jsonResponse({ ...ADA, id: '10', email: 'grace@example.com' }),
    ),
    route<SiteEventsListData>('GET', '/workspaces/{slug}/events', SLUG, (req) => {
      const { searchParams } = new URL(req.url)
      eventRequests.push({
        contactId: searchParams.get('contactId'),
        page: searchParams.get('page'),
      })
      return jsonResponse({
        items: [{ id: 'e1', subjectId: 's1', action: 'page.viewed', createdAt: NOW }],
        totalItems: 25,
      })
    }),
  ])

  // A real in-memory router (renderWithRouter stubs navigation) so the same page
  // instance sees its contactId param change.
  const rootRoute = createRootRoute()
  const detailRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: DETAIL_ROUTE.path,
    component: ContactDetailPage,
  })
  const router = createRouter({
    routeTree: rootRoute.addChildren([detailRoute]),
    history: createMemoryHistory({ initialEntries: [DETAIL_ROUTE.initialPath] }),
  })
  const screen = await renderWithProviders(<RouterProvider router={router} />)

  await expect.element(screen.getByRole('heading', { name: 'ada@example.com' })).toBeInTheDocument()
  await screen.getByRole('button', { name: '2', exact: true }).click()
  await expect.poll(() => eventRequests.at(-1)).toEqual({ contactId: '9', page: '2' })

  await router.navigate({
    to: contactsDetailRoute.to,
    params: { slug: 'test', contactId: '10' },
  })

  await expect
    .element(screen.getByRole('heading', { name: 'grace@example.com' }))
    .toBeInTheDocument()
  await expect.poll(() => eventRequests.at(-1)).toEqual({ contactId: '10', page: '1' })
})
