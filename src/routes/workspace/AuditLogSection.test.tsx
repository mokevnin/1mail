import { expect, test } from 'vitest'

import type { SiteAuditEntryResource, SiteAuditListData } from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { AuditLogSection } from './AuditLogSection.tsx'

const SLUG = 'test'

function entry(over: Partial<SiteAuditEntryResource> = {}): SiteAuditEntryResource {
  return {
    id: '2',
    occurredAt: '2026-01-02T03:04:05Z',
    actor: { kind: 'user', id: '1', name: 'John' },
    action: 'membership.update',
    target: { type: 'membership', id: '3', name: 'Mary' },
    diff: { role: { from: 'member', to: 'admin' } },
    ...over,
  }
}

const list = (respond: () => Response) =>
  route<SiteAuditListData>('GET', '/workspaces/{slug}/audit-entries', { slug: SLUG }, respond)

test('lists who changed what, with the change', async () => {
  mockClientRoutes([list(() => jsonResponse({ items: [entry()] }))])
  const { screen } = await renderWithRouter(<AuditLogSection slug={SLUG} />)

  await expect.element(screen.getByText('Audit log')).toBeInTheDocument()
  await expect.element(screen.getByText('John', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('membership.update')).toBeInTheDocument()
  await expect.element(screen.getByText('membership Mary')).toBeInTheDocument()
  await expect.element(screen.getByText(/"from":"member"/)).toBeInTheDocument()
})

test('shows the empty state', async () => {
  mockClientRoutes([list(() => jsonResponse({ items: [] }))])
  const { screen } = await renderWithRouter(<AuditLogSection slug={SLUG} />)

  await expect.element(screen.getByText('No audited changes yet')).toBeInTheDocument()
})

test('offers the next page while a cursor is returned', async () => {
  mockClientRoutes([list(() => jsonResponse({ items: [entry()], nextCursor: '2' }))])
  const { screen } = await renderWithRouter(<AuditLogSection slug={SLUG} />)

  await expect.element(screen.getByRole('button', { name: 'Load more' })).toBeInTheDocument()
})

test.each([402, 403])('renders nothing when the API answers %i', async (status) => {
  mockClientRoutes([list(() => jsonResponse({ status, detail: 'refused' }, { status }))])
  const { screen } = await renderWithRouter(<AuditLogSection slug={SLUG} />)

  await expect.element(screen.getByText('Audit log')).not.toBeInTheDocument()
})
