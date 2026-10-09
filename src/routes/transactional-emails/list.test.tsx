import { expect, test } from 'vitest'

import type { SiteTransactionalEmailsListData } from '../../generated/site/types.gen.ts'
import { transactionalEmailsRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { formatDateTime } from '../../utils/datetime.ts'
import { TransactionalEmailsListPage } from './list.tsx'

const SLUG = { slug: 'test' }
const LIST_ROUTE = routeMount(transactionalEmailsRoute, SLUG)

const SENT = {
  id: '1',
  destination: 'alice@example.com',
  status: 'sent',
  createdAt: '2026-03-04T10:15:00Z',
}
const FAILED = {
  id: '2',
  destination: 'bob@example.com',
  status: 'failed',
  error: 'mailbox full',
  createdAt: '2026-03-05T11:30:00Z',
}

function listRoute(respond: () => Response) {
  return route<SiteTransactionalEmailsListData>(
    'GET',
    '/workspaces/{slug}/transactional-emails',
    SLUG,
    respond,
  )
}

test('lists transactional emails with status, error and sent time', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [SENT, FAILED], totalItems: 2 }))])

  const { screen } = await renderWithRouter(<TransactionalEmailsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Transactional emails')).toBeInTheDocument()
  await expect.element(screen.getByText('alice@example.com')).toBeInTheDocument()
  await expect.element(screen.getByText('bob@example.com')).toBeInTheDocument()
  await expect.element(screen.getByText('Sent', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('Failed', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('mailbox full')).toBeInTheDocument()
  await expect.element(screen.getByText(formatDateTime(SENT.createdAt))).toBeInTheDocument()
})

test('shows the empty message when there are no emails', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ items: [], totalItems: 0 }))])

  const { screen } = await renderWithRouter(<TransactionalEmailsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('No transactional emails yet')).toBeInTheDocument()
})

test('shows an error alert when the list fails to load', async () => {
  mockClientRoutes([listRoute(() => jsonResponse({ title: 'Boom', status: 500 }, { status: 500 }))])

  const { screen } = await renderWithRouter(<TransactionalEmailsListPage />, LIST_ROUTE)

  await expect.element(screen.getByText('Failed to load transactional emails')).toBeInTheDocument()
})
