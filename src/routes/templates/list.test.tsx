import { expect, test } from 'vitest'

import type {
  SiteTemplatesDeleteData,
  SiteTemplatesListData,
} from '../../generated/site/types.gen.ts'
import { templatesRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { TemplatesListPage } from './list.tsx'

const SLUG = { slug: 'test' }
const LIST_ROUTE = routeMount(templatesRoute, SLUG)

const WELCOME = { id: '1', name: 'Welcome', subject: 'Hello there' }
const RECEIPT = { id: '2', name: 'Receipt', subject: 'Your receipt' }

function listRoute(respond: () => Response) {
  return route<SiteTemplatesListData>('GET', '/workspaces/{slug}/templates', SLUG, respond)
}

test('lists the workspace templates', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [WELCOME, RECEIPT], totalItems: 2 }))])

  const { screen } = await renderWithRouter(<TemplatesListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Welcome')).toBeInTheDocument()
  await expect.element(screen.getByText('Hello there')).toBeInTheDocument()
  await expect.element(screen.getByText('Receipt', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('Your receipt')).toBeInTheDocument()
})

test('shows the empty state when there are no templates', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [], totalItems: 0 }))])

  const { screen } = await renderWithRouter(<TemplatesListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Create your first template.')).toBeInTheDocument()
})

test('shows an error alert when the list fails to load', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ title: 'Boom', status: 500 }, { status: 500 }))])

  const { screen } = await renderWithRouter(<TemplatesListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Failed to load templates')).toBeInTheDocument()
})

test('New template navigates to the create page', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [], totalItems: 0 }))])

  const { screen, navigate } = await renderWithRouter(<TemplatesListPage />, LIST_ROUTE)
  await screen.getByRole('button', { name: 'New template' }).click()

  expect(navigate).toHaveBeenCalledWith(expect.objectContaining({ params: SLUG }))
})

test('Edit navigates to the template edit page', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [WELCOME], totalItems: 1 }))])

  const { screen, navigate } = await renderWithRouter(<TemplatesListPage />, LIST_ROUTE)
  await expect.element(screen.getByText('Welcome')).toBeInTheDocument()
  await screen.getByRole('button', { name: 'Edit' }).click()

  expect(navigate).toHaveBeenCalledWith(
    expect.objectContaining({ params: { slug: 'test', templateId: '1' } }),
  )
})

test('deleting a template asks for confirmation, deletes it and refreshes the list', async () => {
  const deleted: string[] = []
  let listFetches = 0
  mockClientRoutes([
    listRoute(() => {
      listFetches++
      return jsonResponse({ items: deleted.length ? [RECEIPT] : [WELCOME, RECEIPT], totalItems: 2 })
    }),
    route<SiteTemplatesDeleteData>(
      'DELETE',
      '/workspaces/{slug}/templates/{id}',
      { ...SLUG, id: '1' },
      () => {
        deleted.push('1')
        return new Response(null, { status: 204 })
      },
    ),
  ])

  const { screen } = await renderWithRouter(<TemplatesListPage />, LIST_ROUTE)
  await expect.element(screen.getByText('Welcome')).toBeInTheDocument()

  await screen.getByRole('button', { name: 'Delete' }).first().click()
  expect(deleted).toEqual([])
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => deleted).toEqual(['1'])
  await expect.element(screen.getByText('Template deleted')).toBeInTheDocument()
  await expect.poll(() => listFetches).toBe(2)
  await expect.element(screen.getByText('Welcome')).not.toBeInTheDocument()
})
