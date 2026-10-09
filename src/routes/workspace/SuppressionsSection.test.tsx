import { expect, test } from 'vitest'

import type {
  SiteSuppressionResource,
  SiteSuppressionsCreateData,
  SiteSuppressionsDeleteData,
  SiteSuppressionsListData,
} from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { SuppressionsSection } from './SuppressionsSection.tsx'

const SLUG = 'test'

function suppression(over: Partial<SiteSuppressionResource> = {}): SiteSuppressionResource {
  return {
    id: '1',
    channel: 'email',
    destination: 'bounced@example.com',
    reason: 'bounce',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...over,
  }
}

const list = (items: SiteSuppressionResource[], totalItems = items.length) =>
  route<SiteSuppressionsListData>('GET', '/workspaces/{slug}/suppressions', { slug: SLUG }, () =>
    jsonResponse({ items, page: 1, pageSize: 20, totalItems, totalPages: 1 }),
  )

test('lists suppressed addresses with their reason', async () => {
  mockClientRoutes([
    list([
      suppression(),
      suppression({ id: '2', destination: 'angry@example.com', reason: 'complaint' }),
      suppression({ id: '3', destination: 'manual@example.com', reason: 'manual' }),
    ]),
  ])
  const { screen } = await renderWithRouter(<SuppressionsSection slug={SLUG} />)

  await expect.element(screen.getByText('bounced@example.com')).toBeInTheDocument()
  await expect.element(screen.getByText('Bounced', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('Complaint', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('Manual', { exact: true })).toBeInTheDocument()
})

test('adds a suppression', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    list([]),
    route<SiteSuppressionsCreateData>(
      'POST',
      '/workspaces/{slug}/suppressions',
      { slug: SLUG },
      async (req) => {
        bodies.push(await req.json())
        return jsonResponse(suppression({ reason: 'manual' }), { status: 201 })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<SuppressionsSection slug={SLUG} />)

  await screen.getByLabelText(/^Email address/).fill('blocked@example.com')
  await screen.getByRole('button', { name: 'Suppress' }).click()

  await expect.poll(() => bodies).toEqual([{ destination: 'blocked@example.com' }])
})

test('removes a suppression after confirmation', async () => {
  let deleted = false
  mockClientRoutes([
    list([suppression()]),
    route<SiteSuppressionsDeleteData>(
      'DELETE',
      '/workspaces/{slug}/suppressions/{id}',
      { slug: SLUG, id: '1' },
      () => {
        deleted = true
        return new Response(null, { status: 204 })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<SuppressionsSection slug={SLUG} />)

  await screen.getByRole('button', { name: 'Remove' }).click()
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => deleted).toBe(true)
})

test('shows an error alert when the list fails to load', async () => {
  mockClientRoutes([
    route<SiteSuppressionsListData>('GET', '/workspaces/{slug}/suppressions', { slug: SLUG }, () =>
      jsonResponse({ status: 500, detail: 'boom' }, { status: 500 }),
    ),
  ])
  const { screen } = await renderWithRouter(<SuppressionsSection slug={SLUG} />)

  await expect
    .element(screen.getByText('Failed to load suppression list').first())
    .toBeInTheDocument()
})

test('requests the next page when paginating', async () => {
  const pages: (string | null)[] = []
  mockClientRoutes([
    route<SiteSuppressionsListData>(
      'GET',
      '/workspaces/{slug}/suppressions',
      { slug: SLUG },
      (req) => {
        pages.push(new URL(req.url).searchParams.get('page'))
        return jsonResponse({
          items: [suppression()],
          page: 1,
          pageSize: 20,
          totalItems: 45,
          totalPages: 3,
        })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<SuppressionsSection slug={SLUG} />)

  await expect.element(screen.getByText('bounced@example.com')).toBeInTheDocument()
  await screen.getByRole('button', { name: '2', exact: true }).click()

  await expect.poll(() => pages).toContain('2')
})

test('reports a create failure', async () => {
  mockClientRoutes([
    list([]),
    route<SiteSuppressionsCreateData>(
      'POST',
      '/workspaces/{slug}/suppressions',
      { slug: SLUG },
      () => jsonResponse({ status: 422, detail: 'bad address' }, { status: 422 }),
    ),
  ])
  const { screen } = await renderWithRouter(<SuppressionsSection slug={SLUG} />)

  await screen.getByLabelText(/^Email address/).fill('x@example.com')
  await screen.getByRole('button', { name: 'Suppress' }).click()

  await expect.element(screen.getByText('Failed to suppress address')).toBeInTheDocument()
})
