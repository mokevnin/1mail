import { expect, test } from 'vitest'

import type {
  SiteSendingDomainResource,
  SiteSendingDomainsCreateData,
  SiteSendingDomainsDeleteData,
  SiteSendingDomainsListData,
  SiteSendingDomainsVerifyData,
} from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { SendingDomainsSection } from './SendingDomainsSection.tsx'

const SLUG = 'test'

function domain(over: Partial<SiteSendingDomainResource> = {}): SiteSendingDomainResource {
  return {
    id: '1',
    domain: 'mail.acme.com',
    dkimSelector: 'om1',
    verified: false,
    dkimRecord: { type: 'TXT', host: 'om1._domainkey.mail.acme.com', value: 'v=DKIM1; p=KEY' },
    spfRecord: { type: 'TXT', host: 'mail.acme.com', value: 'v=spf1 include:x -all' },
    dmarcRecord: { type: 'TXT', host: '_dmarc.mail.acme.com', value: 'v=DMARC1; p=none' },
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...over,
  }
}

const list = (items: SiteSendingDomainResource[]) =>
  route<SiteSendingDomainsListData>(
    'GET',
    '/workspaces/{slug}/sending-domains',
    { slug: SLUG },
    () => jsonResponse({ items, page: 1, pageSize: 20, totalItems: items.length, totalPages: 1 }),
  )

test('shows pending and verified domains', async () => {
  mockClientRoutes([list([domain(), domain({ id: '2', domain: 'ok.acme.com', verified: true })])])
  const { screen } = await renderWithRouter(<SendingDomainsSection slug={SLUG} />)

  await expect.element(screen.getByText('mail.acme.com')).toBeInTheDocument()
  await expect.element(screen.getByText('Pending', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('Verified', { exact: true })).toBeInTheDocument()
})

test('adds a domain', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    list([]),
    route<SiteSendingDomainsCreateData>(
      'POST',
      '/workspaces/{slug}/sending-domains',
      { slug: SLUG },
      async (req) => {
        bodies.push(await req.json())
        return jsonResponse(domain(), { status: 201 })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<SendingDomainsSection slug={SLUG} />)

  await screen.getByLabelText(/^Domain/).fill(' mail.acme.com ')
  await screen.getByRole('button', { name: 'Add domain' }).click()

  await expect.poll(() => bodies).toEqual([{ domain: 'mail.acme.com' }])
})

test('starts verification', async () => {
  let verified = false
  mockClientRoutes([
    list([domain()]),
    route<SiteSendingDomainsVerifyData>(
      'POST',
      '/workspaces/{slug}/sending-domains/{id}/verify',
      { slug: SLUG, id: '1' },
      () => {
        verified = true
        return new Response(null, { status: 204 })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<SendingDomainsSection slug={SLUG} />)

  await screen.getByRole('button', { name: 'Verify' }).click()

  await expect.poll(() => verified).toBe(true)
  await expect.element(screen.getByText('Verification started')).toBeInTheDocument()
})

test('deletes a domain after confirmation', async () => {
  let deleted = false
  mockClientRoutes([
    list([domain()]),
    route<SiteSendingDomainsDeleteData>(
      'DELETE',
      '/workspaces/{slug}/sending-domains/{id}',
      { slug: SLUG, id: '1' },
      () => {
        deleted = true
        return new Response(null, { status: 204 })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<SendingDomainsSection slug={SLUG} />)

  await screen.getByRole('button', { name: 'Delete' }).click()
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => deleted).toBe(true)
})

test('shows an error alert when the list fails to load', async () => {
  mockClientRoutes([
    route<SiteSendingDomainsListData>(
      'GET',
      '/workspaces/{slug}/sending-domains',
      { slug: SLUG },
      () => jsonResponse({ status: 500, detail: 'boom' }, { status: 500 }),
    ),
  ])
  const { screen } = await renderWithRouter(<SendingDomainsSection slug={SLUG} />)

  await expect
    .element(screen.getByText('Failed to load sending domains').first())
    .toBeInTheDocument()
})

test('expanding a domain row shows its DNS records', async () => {
  mockClientRoutes([list([domain()])])
  const { screen } = await renderWithRouter(<SendingDomainsSection slug={SLUG} />)

  await screen.getByText('mail.acme.com').click()

  await expect.element(screen.getByText('om1._domainkey.mail.acme.com')).toBeInTheDocument()
  await expect.element(screen.getByText('v=DMARC1; p=none')).toBeInTheDocument()
})
