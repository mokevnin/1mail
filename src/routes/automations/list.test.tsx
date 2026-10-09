import { notifications } from '@mantine/notifications'
import { afterEach, expect, test } from 'vitest'

import type {
  SiteAutomationResource,
  SiteAutomationsActivateData,
  SiteAutomationsDeactivateData,
  SiteAutomationsDeleteData,
  SiteAutomationsListData,
} from '../../generated/site/types.gen.ts'
import { automationsRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { AutomationsListPage } from './list.tsx'

const SLUG = { slug: 'test' }
const LIST_ROUTE = routeMount(automationsRoute, SLUG)
const NOW = '2026-01-01T00:00:00Z'

const WELCOME: SiteAutomationResource = {
  id: '1',
  name: 'Welcome flow',
  status: 'active',
  triggerEvent: 'contact.created',
  steps: [],
  createdAt: NOW,
  updatedAt: NOW,
}
const NURTURE: SiteAutomationResource = {
  ...WELCOME,
  id: '2',
  name: 'Nurture flow',
  status: 'draft',
  triggerEvent: 'email.opened',
}

// Toasts live in a global store and would cover the table's buttons in the next test.
afterEach(() => {
  notifications.clean()
})

function listRoute(respond: () => Response) {
  return route<SiteAutomationsListData>('GET', '/workspaces/{slug}/automations', SLUG, respond)
}

test('lists automations with their status', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [WELCOME, NURTURE], totalItems: 2 }))])

  const { screen } = await renderWithRouter(<AutomationsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Welcome flow')).toBeInTheDocument()
  await expect.element(screen.getByText('Nurture flow')).toBeInTheDocument()
  await expect.element(screen.getByText('Active', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('Draft', { exact: true })).toBeInTheDocument()
})

test('shows the empty state when there are no automations', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [], totalItems: 0 }))])

  const { screen } = await renderWithRouter(<AutomationsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Create your first automation.')).toBeInTheDocument()
})

test('shows an error alert when the list fails to load', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ title: 'Boom', status: 500 }, { status: 500 }))])

  const { screen } = await renderWithRouter(<AutomationsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Failed to load automations').first()).toBeInTheDocument()
})

test('New automation and Edit navigate to the create and edit pages', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [WELCOME], totalItems: 1 }))])

  const { screen, navigate } = await renderWithRouter(<AutomationsListPage />, LIST_ROUTE)
  await expect.element(screen.getByText('Welcome flow')).toBeInTheDocument()

  await screen.getByRole('button', { name: 'New automation' }).click()
  expect(navigate).toHaveBeenLastCalledWith(expect.objectContaining({ params: { slug: 'test' } }))

  await screen.getByRole('button', { name: 'Edit' }).click()
  expect(navigate).toHaveBeenLastCalledWith(
    expect.objectContaining({ params: { slug: 'test', automationId: '1' } }),
  )
})

test('activating a draft automation calls the API and refreshes the list', async () => {
  const activated: string[] = []
  let listFetches = 0
  mockClientRoutes([
    listRoute(() => {
      listFetches++
      return jsonResponse({ items: [NURTURE], totalItems: 1 })
    }),
    route<SiteAutomationsActivateData>(
      'POST',
      '/workspaces/{slug}/automations/{id}/activate',
      { ...SLUG, id: '2' },
      () => {
        activated.push('2')
        return jsonResponse({ ...NURTURE, status: 'active' })
      },
    ),
  ])

  const { screen } = await renderWithRouter(<AutomationsListPage />, LIST_ROUTE)
  await screen.getByRole('button', { name: 'Activate' }).click()

  await expect.poll(() => activated).toEqual(['2'])
  await expect.element(screen.getByText('Automation activated')).toBeInTheDocument()
  await expect.poll(() => listFetches).toBe(2)
})

test('deactivating an active automation calls the API', async () => {
  const deactivated: string[] = []
  mockClientRoutes([
    listRoute(() => jsonResponse({ items: [WELCOME], totalItems: 1 })),
    route<SiteAutomationsDeactivateData>(
      'POST',
      '/workspaces/{slug}/automations/{id}/deactivate',
      { ...SLUG, id: '1' },
      () => {
        deactivated.push('1')
        return jsonResponse({ ...WELCOME, status: 'draft' })
      },
    ),
  ])

  const { screen } = await renderWithRouter(<AutomationsListPage />, LIST_ROUTE)
  await screen.getByRole('button', { name: 'Deactivate' }).click()

  await expect.poll(() => deactivated).toEqual(['1'])
  await expect.element(screen.getByText('Automation deactivated')).toBeInTheDocument()
})

test('deleting an automation asks for confirmation first', async () => {
  const deleted: string[] = []
  mockClientRoutes([
    listRoute(() => jsonResponse({ items: [WELCOME], totalItems: 1 })),
    route<SiteAutomationsDeleteData>(
      'DELETE',
      '/workspaces/{slug}/automations/{id}',
      { ...SLUG, id: '1' },
      () => {
        deleted.push('1')
        return new Response(null, { status: 204 })
      },
    ),
  ])

  const { screen } = await renderWithRouter(<AutomationsListPage />, LIST_ROUTE)
  await expect.element(screen.getByText('Welcome flow')).toBeInTheDocument()

  await screen.getByRole('button', { name: 'Delete' }).click()
  expect(deleted).toEqual([])
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => deleted).toEqual(['1'])
  await expect.element(screen.getByText('Automation deleted')).toBeInTheDocument()
})
