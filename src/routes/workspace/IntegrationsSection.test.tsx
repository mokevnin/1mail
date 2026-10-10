import { expect, test } from 'vitest'

import type {
  SiteIntegrationResource,
  SiteIntegrationsCreateData,
  SiteIntegrationsDeleteData,
  SiteIntegrationsListData,
} from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { IntegrationsSection } from './IntegrationsSection.tsx'

const SLUG = 'test'

const smtp: SiteIntegrationResource = {
  id: '1',
  name: 'Primary SMTP',
  channel: 'email',
  provider: 'smtp',
  enabled: true,
  isDefault: true,
  maxPerSecond: null,
  maxPerDay: null,
  config: { kind: 'smtp', host: 'smtp.test', port: 587, from: 'a@acme.com' },
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}

const list = (items: SiteIntegrationResource[]) =>
  route<SiteIntegrationsListData>('GET', '/workspaces/{slug}/integrations', { slug: SLUG }, () =>
    jsonResponse(items),
  )

const create = (bodies: unknown[]) =>
  route<SiteIntegrationsCreateData>(
    'POST',
    '/workspaces/{slug}/integrations',
    { slug: SLUG },
    async (req) => {
      bodies.push(await req.json())
      return jsonResponse(smtp, { status: 201 })
    },
  )

test('lists connected providers', async () => {
  mockClientRoutes([list([smtp])])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await expect.element(screen.getByText('Primary SMTP')).toBeInTheDocument()
})

test('connects an SMTP provider', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([list([]), create(bodies)])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await screen.getByLabelText(/^Name/).fill(' Mailer ')
  await screen.getByLabelText(/^SMTP host/).fill('smtp.acme.com')
  await screen.getByLabelText(/^From address/).fill('hi@acme.com')
  await screen.getByLabelText('Set as default').click()
  await screen.getByRole('button', { name: 'Connect provider' }).click()

  await expect
    .poll(() => bodies)
    .toEqual([
      {
        name: 'Mailer',
        isDefault: true,
        config: {
          kind: 'smtp',
          host: 'smtp.acme.com',
          port: 587,
          username: null,
          password: null,
          from: 'hi@acme.com',
          fromName: null,
        },
      },
    ])
})

test('connects an Amazon SES provider', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([list([]), create(bodies)])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await screen.getByRole('combobox', { name: 'Provider' }).click()
  await screen.getByRole('option', { name: 'Amazon SES' }).click()
  await screen.getByLabelText(/^Name/).fill('SES')
  await screen.getByLabelText(/^AWS region/).fill('eu-west-1')
  await screen.getByLabelText(/^Access key ID/).fill('AKIA')
  await screen.getByLabelText(/^Secret access key/).fill('shh')
  await screen.getByLabelText(/^From address/).fill('hi@acme.com')
  await screen.getByLabelText(/^From name/).fill('Acme')
  await screen.getByRole('button', { name: 'Connect provider' }).click()

  await expect
    .poll(() => bodies)
    .toEqual([
      {
        name: 'SES',
        isDefault: false,
        config: {
          kind: 'ses',
          region: 'eu-west-1',
          accessKeyId: 'AKIA',
          secretAccessKey: 'shh',
          endpoint: null,
          from: 'hi@acme.com',
          fromName: 'Acme',
        },
      },
    ])
})

test('deletes a provider after confirmation', async () => {
  let deleted = false
  mockClientRoutes([
    list([smtp]),
    route<SiteIntegrationsDeleteData>(
      'DELETE',
      '/workspaces/{slug}/integrations/{id}',
      { slug: SLUG, id: '1' },
      () => {
        deleted = true
        return new Response(null, { status: 204 })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await screen.getByRole('button', { name: 'Delete' }).click()
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => deleted).toBe(true)
})

test('shows an error alert when providers fail to load', async () => {
  mockClientRoutes([
    route<SiteIntegrationsListData>('GET', '/workspaces/{slug}/integrations', { slug: SLUG }, () =>
      jsonResponse({ status: 500, detail: 'boom' }, { status: 500 }),
    ),
  ])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await expect.element(screen.getByText('Failed to load providers').first()).toBeInTheDocument()
})

test('reports a create failure', async () => {
  mockClientRoutes([
    list([]),
    route<SiteIntegrationsCreateData>(
      'POST',
      '/workspaces/{slug}/integrations',
      { slug: SLUG },
      () => jsonResponse({ status: 422, detail: 'bad host' }, { status: 422 }),
    ),
  ])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await screen.getByLabelText(/^Name/).fill('X')
  await screen.getByLabelText(/^SMTP host/).fill('h')
  await screen.getByLabelText(/^From address/).fill('a@b.co')
  await screen.getByRole('button', { name: 'Connect provider' }).click()

  await expect.element(screen.getByText('Failed to save provider')).toBeInTheDocument()
})
