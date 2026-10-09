import { expect, test } from 'vitest'

import type {
  SiteWebhookEndpointResource,
  SiteWebhooksCreateData,
  SiteWebhooksDeleteData,
  SiteWebhooksListData,
  SiteWebhooksUpdateData,
} from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { WebhooksSection } from './WebhooksSection.tsx'

const SLUG = 'test'

function endpoint(over: Partial<SiteWebhookEndpointResource> = {}): SiteWebhookEndpointResource {
  return {
    id: '1',
    url: 'https://example.com/hook',
    secret: 'whsec_1',
    eventTypes: [],
    enabled: true,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...over,
  }
}

function page(items: SiteWebhookEndpointResource[]) {
  return jsonResponse({ items, page: 1, pageSize: 20, totalItems: items.length, totalPages: 1 })
}

const list = (items: SiteWebhookEndpointResource[]) =>
  route<SiteWebhooksListData>('GET', '/workspaces/{slug}/webhooks', { slug: SLUG }, () =>
    page(items),
  )

test('lists endpoints, showing all events and the enabled state', async () => {
  mockClientRoutes([
    list([
      endpoint(),
      endpoint({
        id: '2',
        url: 'https://example.com/b',
        eventTypes: ['email.opened'],
        enabled: false,
      }),
    ]),
  ])
  const { screen } = await renderWithRouter(<WebhooksSection slug={SLUG} />)

  await expect.element(screen.getByText('https://example.com/hook')).toBeInTheDocument()
  await expect.element(screen.getByText('All events', { exact: true })).toBeInTheDocument()
  await expect
    .element(screen.getByRole('cell', { name: 'email.opened', exact: true }))
    .toBeInTheDocument()
  await expect.element(screen.getByText('Disabled', { exact: true })).toBeInTheDocument()
})

test('creates an endpoint from the form', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    list([]),
    route<SiteWebhooksCreateData>(
      'POST',
      '/workspaces/{slug}/webhooks',
      { slug: SLUG },
      async (req) => {
        bodies.push(await req.json())
        return jsonResponse(endpoint(), { status: 201 })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<WebhooksSection slug={SLUG} />)

  await screen.getByLabelText(/^Endpoint URL/).fill('  https://example.com/new  ')
  await screen.getByRole('button', { name: 'Add endpoint' }).click()

  await expect.poll(() => bodies).toEqual([{ url: 'https://example.com/new', eventTypes: [] }])
})

test('toggles an endpoint off', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    list([endpoint({ eventTypes: ['email.opened'] })]),
    route<SiteWebhooksUpdateData>(
      'PUT',
      '/workspaces/{slug}/webhooks/{id}',
      { slug: SLUG, id: '1' },
      async (req) => {
        bodies.push(await req.json())
        return jsonResponse(endpoint({ enabled: false }))
      },
    ),
  ])
  const { screen } = await renderWithRouter(<WebhooksSection slug={SLUG} />)

  await screen.getByRole('button', { name: 'Disable' }).click()

  await expect
    .poll(() => bodies)
    .toEqual([{ url: 'https://example.com/hook', eventTypes: ['email.opened'], enabled: false }])
})

test('deletes an endpoint after confirmation', async () => {
  let deleted = false
  mockClientRoutes([
    list([endpoint()]),
    route<SiteWebhooksDeleteData>(
      'DELETE',
      '/workspaces/{slug}/webhooks/{id}',
      { slug: SLUG, id: '1' },
      () => {
        deleted = true
        return new Response(null, { status: 204 })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<WebhooksSection slug={SLUG} />)

  await screen.getByRole('button', { name: 'Delete' }).click()
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => deleted).toBe(true)
})

test('shows an error alert when the list fails to load', async () => {
  mockClientRoutes([
    route<SiteWebhooksListData>('GET', '/workspaces/{slug}/webhooks', { slug: SLUG }, () =>
      jsonResponse({ status: 500, detail: 'boom' }, { status: 500 }),
    ),
  ])
  const { screen } = await renderWithRouter(<WebhooksSection slug={SLUG} />)

  await expect.element(screen.getByText('Failed to load webhooks').first()).toBeInTheDocument()
})

test('reports a create failure', async () => {
  mockClientRoutes([
    list([]),
    route<SiteWebhooksCreateData>('POST', '/workspaces/{slug}/webhooks', { slug: SLUG }, () =>
      jsonResponse({ status: 422, detail: 'url must be https' }, { status: 422 }),
    ),
  ])
  const { screen } = await renderWithRouter(<WebhooksSection slug={SLUG} />)

  await screen.getByLabelText(/^Endpoint URL/).fill('http://x.test')
  await screen.getByRole('button', { name: 'Add endpoint' }).click()

  await expect.element(screen.getByText('Failed to create webhook')).toBeInTheDocument()
})
