import { expect, test } from 'vitest'

import type {
  SiteIntegrationResource,
  SiteIntegrationsCreateData,
  SiteIntegrationsDeleteData,
  SiteIntegrationsListData,
  SiteIntegrationsUpdateData,
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
  sendLimit: {
    perSecond: { limit: null, source: null },
    perDay: { limit: null, source: null },
    sentLast24h: 0,
    warnings: ['unlimited'],
  },
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
        maxPerSecond: null,
        maxPerDay: null,
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
        maxPerSecond: null,
        maxPerDay: null,
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

const limited: SiteIntegrationResource = {
  ...smtp,
  maxPerSecond: 14,
  maxPerDay: 50000,
  sendLimit: {
    perSecond: { limit: 14, source: 'manual' },
    perDay: { limit: 50000, source: 'manual' },
    sentLast24h: 1200,
    warnings: [],
  },
}

const update = (bodies: unknown[]) =>
  route<SiteIntegrationsUpdateData>(
    'PUT',
    '/workspaces/{slug}/integrations/{id}',
    { slug: SLUG, id: '1' },
    async (req) => {
      bodies.push(await req.json())
      return jsonResponse(limited)
    },
  )

test('warns when an Integration has no send limit', async () => {
  mockClientRoutes([list([smtp])])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await expect.element(screen.getByText('Primary SMTP has no send limit')).toBeInTheDocument()
  await expect.element(screen.getByText('Per second: Unlimited')).toBeInTheDocument()
})

test('warns when the provider send quota could not be read', async () => {
  const ses: SiteIntegrationResource = {
    ...limited,
    name: 'Amazon SES',
    sendLimit: { ...limited.sendLimit, warnings: ['providerQuotaUnavailable'] },
  }
  mockClientRoutes([list([ses])])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await expect
    .element(screen.getByText("Amazon SES: the provider's send quota could not be read"))
    .toBeInTheDocument()
  await expect
    .element(screen.getByText('Without the provider quota', { exact: false }))
    .toBeInTheDocument()
})

test('shows no quota warning when the provider quota is known', async () => {
  mockClientRoutes([list([limited])])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await expect.element(screen.getByText('Per second: 14')).toBeInTheDocument()
  await expect
    .element(screen.getByText('send quota could not be read', { exact: false }))
    .not.toBeInTheDocument()
})

test('shows the effective limit, its source and the 24-hour usage', async () => {
  mockClientRoutes([list([limited])])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await expect.element(screen.getByText('Per second: 14')).toBeInTheDocument()
  await expect.element(screen.getByText('Per 24 h: 50 000')).toBeInTheDocument()
  await expect
    .element(screen.getByText('Sent in the last 24 h: 1 200 / 50 000'))
    .toBeInTheDocument()
  await expect.element(screen.getByText('Manual').first()).toBeInTheDocument()
  await expect.element(screen.getByText('has no send limit')).not.toBeInTheDocument()
})

test('connects a provider with send limits', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([list([]), create(bodies)])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await screen.getByLabelText(/^Name/).fill('Mailer')
  await screen.getByLabelText(/^SMTP host/).fill('smtp.acme.com')
  await screen.getByLabelText(/^From address/).fill('hi@acme.com')
  await screen.getByLabelText('Max per second').fill('5')
  await screen.getByLabelText('Max per 24 hours').fill('1000')
  await screen.getByRole('button', { name: 'Connect provider' }).click()

  await expect
    .poll(() => bodies)
    .toMatchObject([{ name: 'Mailer', maxPerSecond: 5, maxPerDay: 1000 }])
})

test('edits only the limits of an existing provider, and a blank clears one', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([list([limited]), update(bodies)])
  const { screen } = await renderWithRouter(<IntegrationsSection slug={SLUG} />)

  await screen.getByRole('button', { name: 'Limits' }).click()
  const dialog = screen.getByRole('dialog')
  await expect.element(dialog.getByLabelText('Max per second')).toHaveValue('14')
  await dialog.getByLabelText('Max per second').fill('20')
  await dialog.getByLabelText('Max per 24 hours').clear()
  await dialog.getByRole('button', { name: 'Save limits' }).click()

  await expect.poll(() => bodies).toEqual([{ maxPerSecond: 20, maxPerDay: null }])
})
