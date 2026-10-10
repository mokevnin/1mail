import { expect, test, vi } from 'vitest'

import type {
  SiteAuditEntryResource,
  SiteAuditExportData,
  SiteAuditListData,
} from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import type { AuditFilter } from './auditFilter.ts'
import { AuditLogSection, ChangeHistoryLink } from './AuditLogSection.tsx'

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
  const { screen } = await renderWithRouter(
    <AuditLogSection slug={SLUG} onFilterChange={() => {}} />,
  )

  await expect.element(screen.getByText('Audit log')).toBeInTheDocument()
  await expect.element(screen.getByText('John', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('membership.update')).toBeInTheDocument()
  await expect.element(screen.getByText('membership Mary')).toBeInTheDocument()
})

test('expands an entry to its before/after diff', async () => {
  mockClientRoutes([list(() => jsonResponse({ items: [entry()] }))])
  const { screen } = await renderWithRouter(
    <AuditLogSection slug={SLUG} onFilterChange={() => {}} />,
  )

  await expect.element(screen.getByText('admin', { exact: true })).not.toBeInTheDocument()
  await screen.getByRole('button', { name: 'Show changes' }).click()
  await expect.element(screen.getByText('role', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('member', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('admin', { exact: true })).toBeInTheDocument()
})

test('shows a sensitive field as changed only', async () => {
  mockClientRoutes([list(() => jsonResponse({ items: [entry({ diff: { secret: 'changed' } })] }))])
  const { screen } = await renderWithRouter(
    <AuditLogSection slug={SLUG} onFilterChange={() => {}} />,
  )

  await screen.getByRole('button', { name: 'Show changes' }).click()
  await expect.element(screen.getByText('secret', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('changed', { exact: true })).toBeInTheDocument()
})

test('sends the applied filter to the list and to the export', async () => {
  const queries: string[] = []
  let exported = ''
  mockClientRoutes([
    route<SiteAuditListData>('GET', '/workspaces/{slug}/audit-entries', { slug: SLUG }, (req) => {
      queries.push(new URL(req.url).search)
      return jsonResponse({ items: [entry()] })
    }),
    route<SiteAuditExportData>(
      'GET',
      '/workspaces/{slug}/audit-entries/export',
      { slug: SLUG },
      (req) => {
        exported = new URL(req.url).search
        return new Response('id\n', { headers: { 'content-type': 'text/csv' } })
      },
    ),
  ])
  const { screen } = await renderWithRouter(
    <AuditLogSection
      slug={SLUG}
      filter={{ targetType: 'integration', targetId: '5', ip: '10.0.0.1' }}
      onFilterChange={() => {}}
    />,
  )

  await vi.waitFor(() =>
    expect(
      queries.some((q) => q.includes('targetType=integration') && q.includes('targetId=5')),
    ).toBe(true),
  )
  expect(queries.some((q) => q.includes('ip=10.0.0.1'))).toBe(true)
  await expect.element(screen.getByLabelText('Target type')).toHaveValue('integration')

  await screen.getByRole('button', { name: 'Export CSV' }).click()
  await vi.waitFor(() => expect(exported).toContain('targetType=integration'))
  expect(exported).toContain('targetId=5')
})

test('applying the form hands the new filter up, and reset clears it', async () => {
  mockClientRoutes([list(() => jsonResponse({ items: [entry()] }))])
  const onFilterChange = vi.fn<(filter: AuditFilter) => void>()
  const { screen } = await renderWithRouter(
    <AuditLogSection slug={SLUG} onFilterChange={onFilterChange} />,
  )

  await screen.getByLabelText('Request id').fill(' req-1 ')
  await screen.getByLabelText('Target type').fill('webhook_endpoint')
  await screen.getByRole('button', { name: 'Apply filters' }).click()
  expect(onFilterChange).toHaveBeenCalledWith({
    requestId: 'req-1',
    targetType: 'webhook_endpoint',
  })

  await screen.getByRole('button', { name: 'Reset filters' }).click()
  expect(onFilterChange).toHaveBeenLastCalledWith({})
})

test('says so when the filter matches nothing', async () => {
  mockClientRoutes([list(() => jsonResponse({ items: [] }))])
  const { screen } = await renderWithRouter(
    <AuditLogSection slug={SLUG} filter={{ ip: '1.2.3.4' }} onFilterChange={() => {}} />,
  )

  await expect.element(screen.getByText('No entries match the filters')).toBeInTheDocument()
})

test('the change history link opens the log filtered to the object', async () => {
  mockClientRoutes([list(() => jsonResponse({ items: [] }))])
  const { screen } = await renderWithRouter(
    <ChangeHistoryLink slug={SLUG} targetType="integration" targetId="5" />,
  )

  const link = screen.getByRole('link', { name: 'Change history' })
  await expect.element(link).toBeInTheDocument()
  const href = link.element().getAttribute('href') ?? ''
  expect(href).toContain('targetType=integration')
  expect(href).toContain('targetId=%225%22')
})

test('the change history link is not offered without the log', async () => {
  mockClientRoutes([list(() => jsonResponse({ status: 402, detail: 'no' }, { status: 402 }))])
  const { screen } = await renderWithRouter(
    <ChangeHistoryLink slug={SLUG} targetType="integration" targetId="5" />,
  )

  await expect.element(screen.getByRole('link', { name: 'Change history' })).not.toBeInTheDocument()
})

test('shows the empty state', async () => {
  mockClientRoutes([list(() => jsonResponse({ items: [] }))])
  const { screen } = await renderWithRouter(
    <AuditLogSection slug={SLUG} onFilterChange={() => {}} />,
  )

  await expect.element(screen.getByText('No audited changes yet')).toBeInTheDocument()
})

test('offers the next page while a cursor is returned', async () => {
  mockClientRoutes([list(() => jsonResponse({ items: [entry()], nextCursor: '2' }))])
  const { screen } = await renderWithRouter(
    <AuditLogSection slug={SLUG} onFilterChange={() => {}} />,
  )

  await expect.element(screen.getByRole('button', { name: 'Load more' })).toBeInTheDocument()
})

test.each([402, 403])('renders nothing when the API answers %i', async (status) => {
  mockClientRoutes([list(() => jsonResponse({ status, detail: 'refused' }, { status }))])
  const { screen } = await renderWithRouter(
    <AuditLogSection slug={SLUG} onFilterChange={() => {}} />,
  )

  await expect.element(screen.getByText('Audit log')).not.toBeInTheDocument()
})

test('exports the log as CSV through the generated client', async () => {
  let exported = 0
  mockClientRoutes([
    list(() => jsonResponse({ items: [entry()] })),
    route<SiteAuditExportData>(
      'GET',
      '/workspaces/{slug}/audit-entries/export',
      { slug: SLUG },
      () => {
        exported += 1
        return new Response('id,action\n2,membership.update\n', {
          headers: { 'content-type': 'text/csv' },
        })
      },
    ),
  ])
  const { screen } = await renderWithRouter(
    <AuditLogSection slug={SLUG} onFilterChange={() => {}} />,
  )

  await screen.getByRole('button', { name: 'Export CSV' }).click()
  await vi.waitFor(() => expect(exported).toBe(1))
})
