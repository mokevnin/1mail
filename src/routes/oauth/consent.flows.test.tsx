import { expect, test } from 'vitest'

import type {
  SiteOAuthDecideData,
  SiteOAuthDescribeData,
  SiteWorkspacesListData,
} from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { OAuthConsent } from './consent.tsx'

const request = {
  clientId: 'fixture-client',
  redirectUri: 'https://connector.example/callback',
  codeChallenge: 'a'.repeat(43),
  state: 'xyz',
  scope: 'contacts:read',
}

const workspace = (slug: string, name: string) => ({
  id: slug,
  name,
  slug,
  collectKey: 'omck',
  ingestKey: 'omik',
  postalAddress: '',
  createdAt: '2026-01-01T00:00:00Z',
})

function serve(
  workspaces: ReturnType<typeof workspace>[],
  decide: (req: Request) => Response | Promise<Response>,
) {
  mockClientRoutes([
    route<SiteWorkspacesListData>('GET', '/workspaces', {}, () => jsonResponse(workspaces)),
    route<SiteOAuthDescribeData>('GET', '/oauth/authorization', {}, () =>
      jsonResponse({
        clientName: 'Fixture Connector',
        redirectUri: request.redirectUri,
        scopes: ['contacts:read'],
        sendScopes: [],
      }),
    ),
    route<SiteOAuthDecideData>('POST', '/oauth/authorization', {}, decide),
  ])
}

test('the decision targets the workspace the user picked', async () => {
  const slugs: string[] = []
  serve([workspace('acme', 'Acme'), workspace('beta', 'Beta')], async (req) => {
    slugs.push((await req.json()).workspaceSlug)
    return jsonResponse({ redirectUrl: '#done' })
  })
  const { screen } = await renderWithRouter(<OAuthConsent request={request} />)

  await screen.getByRole('combobox', { name: 'Workspace' }).click()
  await screen.getByRole('option', { name: 'Beta' }).click()
  await screen.getByRole('button', { name: 'Allow access' }).click()

  await expect.poll(() => slugs).toEqual(['beta'])
})

test('without a workspace the user cannot approve or deny', async () => {
  let decided = false
  serve([], () => {
    decided = true
    return jsonResponse({ redirectUrl: '#done' })
  })
  const { screen } = await renderWithRouter(<OAuthConsent request={request} />)

  await expect.element(screen.getByRole('button', { name: 'Allow access' })).toBeDisabled()
  await screen.getByRole('button', { name: 'Deny' }).click()

  expect(decided).toBe(false)
})

test('an unexpected failure is toasted with the API detail', async () => {
  serve([workspace('acme', 'Acme')], () =>
    jsonResponse({ status: 500, detail: 'the code store is down' }, { status: 500 }),
  )
  const { screen } = await renderWithRouter(<OAuthConsent request={request} />)

  await screen.getByRole('button', { name: 'Allow access' }).click()

  await expect.element(screen.getByText('Authorization failed')).toBeInTheDocument()
  await expect.element(screen.getByText('the code store is down')).toBeInTheDocument()
})
