import { expect, test } from 'vitest'

import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { OAuthConsent } from './consent.tsx'

const request = {
  clientId: 'fixture-client',
  redirectUri: 'https://connector.example/callback',
  codeChallenge: 'a'.repeat(43),
  state: 'xyz',
  scope: 'contacts:read emails:send',
}

const workspace = {
  id: '1',
  name: 'Acme',
  slug: 'acme',
  collectKey: 'omck_test_key',
  ingestKey: 'omik_test_key',
  postalAddress: '',
  createdAt: '2026-01-01T00:00:00Z',
}

function mockConsentApi(onDecide: (body: unknown) => void) {
  mockClientFetch(async (input) => {
    const req = input instanceof Request ? input : new Request(String(input))
    if (req.url.includes('/oauth/authorization')) {
      if (req.method === 'POST') {
        onDecide(await req.clone().json())
        // Stay on the page: the test only inspects the decision that was sent.
        return jsonResponse({ redirectUrl: '#done' })
      }
      return jsonResponse({
        clientName: 'Fixture Connector',
        redirectUri: request.redirectUri,
        scopes: ['contacts:read'],
        sendScopes: ['emails:send'],
      })
    }
    return jsonResponse([workspace])
  })
}

test('shows the client and its permissions, with sending off by default', async () => {
  mockConsentApi(() => {})
  const { screen } = await renderWithRouter(<OAuthConsent request={request} />)

  await expect.element(screen.getByText('Connect Fixture Connector to 1mail')).toBeInTheDocument()
  await expect.element(screen.getByText('contacts:read', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByLabelText(/Also allow sending/)).not.toBeChecked()
})

test('approving without the send opt-in does not allow sending', async () => {
  const decisions: unknown[] = []
  mockConsentApi((body) => decisions.push(body))
  const { screen } = await renderWithRouter(<OAuthConsent request={request} />)

  await screen.getByRole('button', { name: 'Allow access' }).click()

  await expect.poll(() => decisions).toHaveLength(1)
  expect(decisions[0]).toMatchObject({
    clientId: 'fixture-client',
    workspaceSlug: 'acme',
    approve: true,
    allowSend: false,
    state: 'xyz',
  })
})

test('ticking the send opt-in is carried to the decision', async () => {
  const decisions: unknown[] = []
  mockConsentApi((body) => decisions.push(body))
  const { screen } = await renderWithRouter(<OAuthConsent request={request} />)

  await screen.getByLabelText(/Also allow sending/).click()
  await screen.getByRole('button', { name: 'Allow access' }).click()

  await expect.poll(() => decisions).toHaveLength(1)
  expect(decisions[0]).toMatchObject({ approve: true, allowSend: true })
})

test('denying sends approve=false', async () => {
  const decisions: unknown[] = []
  mockConsentApi((body) => decisions.push(body))
  const { screen } = await renderWithRouter(<OAuthConsent request={request} />)

  await screen.getByRole('button', { name: 'Deny' }).click()

  await expect.poll(() => decisions).toHaveLength(1)
  expect(decisions[0]).toMatchObject({ approve: false, allowSend: false })
})

test('an invalid request shows an error instead of the form', async () => {
  mockClientFetch((input) => {
    const url = input instanceof Request ? input.url : String(input)
    return url.includes('/oauth/authorization')
      ? jsonResponse({ detail: 'bad' }, { status: 400 })
      : jsonResponse([workspace])
  })
  const { screen } = await renderWithRouter(<OAuthConsent request={request} />)

  await expect.element(screen.getByText(/authorization request is invalid/)).toBeInTheDocument()
})
