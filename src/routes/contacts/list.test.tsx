import { expect, test } from 'vitest'

import type {
  SiteContactsDeleteData,
  SiteContactsListData,
} from '../../generated/site/types.gen.ts'
import { contactsRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { ContactsListPage } from './list.tsx'

const SLUG = { slug: 'test' }
const LIST_ROUTE = routeMount(contactsRoute, SLUG)

const ALICE = { id: '1', email: 'alice@example.com', firstName: 'Alice', lastName: 'Smith' }
const BOB = { id: '2', email: 'bob@example.com' }

function listRoute(respond: () => Response) {
  return route<SiteContactsListData>('GET', '/workspaces/{slug}/contacts', SLUG, respond)
}

test('lists the workspace contacts', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [ALICE, BOB], totalItems: 2 }))])

  const { screen } = await renderWithRouter(<ContactsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('alice@example.com')).toBeInTheDocument()
  await expect.element(screen.getByText('bob@example.com')).toBeInTheDocument()
  await expect.element(screen.getByText('Smith')).toBeInTheDocument()
})

test('shows the empty state when there are no contacts', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [], totalItems: 0 }))])

  const { screen } = await renderWithRouter(<ContactsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Create your first contact.')).toBeInTheDocument()
})

test('shows an error alert when the list fails to load', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ title: 'Boom', status: 500 }, { status: 500 }))])

  const { screen } = await renderWithRouter(<ContactsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Failed to load contacts')).toBeInTheDocument()
})

test('Add contact navigates to the create page', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [], totalItems: 0 }))])

  const { screen, navigate } = await renderWithRouter(<ContactsListPage />, LIST_ROUTE)
  await screen.getByRole('button', { name: 'Add contact' }).click()

  expect(navigate).toHaveBeenCalledWith(expect.objectContaining({ params: { slug: 'test' } }))
})

test('deleting a contact asks for confirmation, deletes it and refreshes the list', async () => {
  const deleted: string[] = []
  let listFetches = 0
  mockClientRoutes([
    listRoute(() => {
      listFetches++
      return jsonResponse({ items: deleted.length ? [BOB] : [ALICE, BOB], totalItems: 2 })
    }),
    route<SiteContactsDeleteData>(
      'DELETE',
      '/workspaces/{slug}/contacts/{id}',
      { ...SLUG, id: '1' },
      () => {
        deleted.push('1')
        return new Response(null, { status: 204 })
      },
    ),
  ])

  const { screen } = await renderWithRouter(<ContactsListPage />, LIST_ROUTE)
  await expect.element(screen.getByText('alice@example.com')).toBeInTheDocument()

  await screen.getByRole('button', { name: 'Delete' }).first().click()
  expect(deleted).toEqual([])
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => deleted).toEqual(['1'])
  await expect.element(screen.getByText('Contact deleted')).toBeInTheDocument()
  await expect.poll(() => listFetches).toBe(2)
  await expect.element(screen.getByText('alice@example.com')).not.toBeInTheDocument()
})

test('changing the page requests page 2 of the contacts', async () => {
  const pages: (string | null)[] = []
  mockClientRoutes([
    route<SiteContactsListData>('GET', '/workspaces/{slug}/contacts', SLUG, (req) => {
      pages.push(new URL(req.url).searchParams.get('page'))
      return jsonResponse({ items: [ALICE], totalItems: 25 })
    }),
  ])

  const { screen } = await renderWithRouter(<ContactsListPage />, LIST_ROUTE)
  await expect.element(screen.getByText('alice@example.com')).toBeInTheDocument()
  await screen.getByRole('button', { name: '2', exact: true }).click()

  await expect.poll(() => pages).toEqual(['1', '2'])
})

test('a failed delete shows the error toast and keeps the contact', async () => {
  mockClientRoutes([
    listRoute(() => jsonResponse({ items: [ALICE], totalItems: 1 })),
    route<SiteContactsDeleteData>(
      'DELETE',
      '/workspaces/{slug}/contacts/{id}',
      { ...SLUG, id: '1' },
      () => jsonResponse({ title: 'Boom', status: 500 }, { status: 500 }),
    ),
  ])

  const { screen } = await renderWithRouter(<ContactsListPage />, LIST_ROUTE)
  await screen.getByRole('button', { name: 'Delete' }).first().click()
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.element(screen.getByText('Failed to delete', { exact: false })).toBeInTheDocument()
  await expect.element(screen.getByText('alice@example.com')).toBeInTheDocument()
})
