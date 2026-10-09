import { useQuery } from '@tanstack/react-query'
import { expect, test } from 'vitest'

import {
  siteContactsGetOptions,
  siteContactsListOptions,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { ContactCreatePage, ContactEditPage } from './resource.tsx'

const CREATE_ROUTE = {
  path: '/workspaces/$slug/contacts/new',
  initialPath: '/workspaces/test/contacts/new',
}

// Stands in for the contacts list screen: an active list query, so invalidation after a save
// is observable as a refetch.
function ListProbe() {
  useQuery(siteContactsListOptions({ path: { slug: 'test' } }))
  return null
}

function requestOf(input: RequestInfo | URL, init?: RequestInit) {
  return input instanceof Request ? input : new Request(String(input), init)
}

test('creating a contact sends only the filled keys, refreshes the list and opens the edit page', async () => {
  const bodies: unknown[] = []
  let listFetches = 0
  mockClientFetch(async (input, init) => {
    const req = requestOf(input, init)
    if (req.method === 'POST') {
      bodies.push(await req.json())
      return jsonResponse({ id: '42', email: 'ada@example.com', firstName: 'Ada' }, { status: 201 })
    }
    listFetches++
    return jsonResponse({ items: [], totalItems: 0 })
  })

  const { screen, navigate } = await renderWithRouter(
    <>
      <ListProbe />
      <ContactCreatePage />
    </>,
    CREATE_ROUTE,
  )
  await expect.poll(() => listFetches).toBe(1)

  await screen.getByLabelText('Email').fill('ada@example.com')
  await screen.getByLabelText('First name').fill('Ada')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.poll(() => bodies).toEqual([{ email: 'ada@example.com', firstName: 'Ada' }])
  await expect.element(screen.getByText('Contact created')).toBeInTheDocument()
  await expect.poll(() => listFetches).toBe(2)
  await expect.poll(() => navigate.mock.calls.length).toBe(1)
  expect(navigate.mock.calls[0]?.[0]).toMatchObject({
    params: { slug: 'test', contactId: '42' },
  })
})

test('a malformed email is rejected before anything is sent', async () => {
  const bodies: unknown[] = []
  mockClientFetch(async (input, init) => {
    bodies.push(await requestOf(input, init).json())
    return jsonResponse({ id: '1' }, { status: 201 })
  })

  const { screen } = await renderWithRouter(<ContactCreatePage />, CREATE_ROUTE)

  await screen.getByLabelText('Email').fill('not-an-email')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.element(screen.getByText('Invalid email address')).toBeVisible()
  expect(bodies).toEqual([])
})

const EDIT_ROUTE = {
  path: '/workspaces/$slug/contacts/$contactId/edit',
  initialPath: '/workspaces/test/contacts/7/edit',
}

const LOADED_CONTACT = {
  id: '7',
  email: 'ada@example.com',
  phone: null,
  firstName: 'Ada',
  customFields: { plan: 'pro' },
}

// Active detail query next to the edit page, so its invalidation is observable.
function DetailProbe() {
  useQuery(siteContactsGetOptions({ path: { slug: 'test', id: '7' } }))
  return null
}

test('editing shows existing values with nulls as blanks; a cleared field is sent as null', async () => {
  const bodies: unknown[] = []
  let detailFetches = 0
  let listFetches = 0
  mockClientFetch(async (input, init) => {
    const req = requestOf(input, init)
    if (req.method === 'PUT' || req.method === 'PATCH') {
      bodies.push(await req.json())
      return jsonResponse({ id: '7', email: 'ada@example.com' })
    }
    if (new URL(req.url, 'http://x').pathname.endsWith('/contacts')) {
      listFetches++
      return jsonResponse({ items: [], totalItems: 0 })
    }
    detailFetches++
    return jsonResponse(LOADED_CONTACT)
  })

  const { screen } = await renderWithRouter(
    <>
      <ListProbe />
      <DetailProbe />
      <ContactEditPage />
    </>,
    EDIT_ROUTE,
  )

  await expect.element(screen.getByLabelText('Email')).toHaveValue('ada@example.com')
  await expect.element(screen.getByLabelText('First name')).toHaveValue('Ada')
  await expect.element(screen.getByLabelText('Phone')).toHaveValue('')

  await screen.getByLabelText('First name').fill('')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect
    .poll(() => bodies)
    .toEqual([
      {
        subjectId: null,
        email: 'ada@example.com',
        phone: null,
        firstName: null,
        lastName: null,
        timeZone: null,
      },
    ])
  await expect.element(screen.getByText('Contact updated')).toBeInTheDocument()
  await expect.poll(() => listFetches).toBe(2)
  await expect.poll(() => detailFetches).toBeGreaterThan(1)
})

test('editing shows an error alert when the contact cannot be loaded', async () => {
  mockClientFetch(() =>
    jsonResponse({ title: 'Not Found', detail: 'contact not found' }, { status: 404 }),
  )

  const { screen } = await renderWithRouter(<ContactEditPage />, EDIT_ROUTE)

  await expect.element(screen.getByText('Failed to load contacts')).toBeVisible()
  await expect.element(screen.getByText('contact not found')).toBeVisible()
})
