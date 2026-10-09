import { expect, test } from 'vitest'

import { indexRoute, loginRoute, oauthConsentRoute, router, workspaceRoute } from './router.tsx'
import { jsonResponse, mockClientFetch } from './test/mockFetch.ts'

const workspace = (slug: string) => ({
  id: slug,
  name: slug,
  slug,
  collectKey: 'omck',
  ingestKey: 'omik',
  postalAddress: '',
  createdAt: '2026-01-01T00:00:00Z',
})

// The guards only ever call the workspaces list, so one stub per scenario is enough.
function serveWorkspaces(slugs: string[] | 'unauthorized') {
  mockClientFetch(() =>
    slugs === 'unauthorized'
      ? jsonResponse({ status: 401, detail: 'unauthorized' }, { status: 401 })
      : jsonResponse(slugs.map(workspace)),
  )
}

test('the index route sends a signed-in user to their first workspace', async () => {
  serveWorkspaces(['acme', 'other'])

  await router.navigate({ to: indexRoute.to })

  expect(router.state.location.pathname).toBe(workspaceRoute.fullPath.replace('$slug', 'acme'))
})

test('the index route sends a user without workspaces to login', async () => {
  serveWorkspaces([])

  await router.navigate({ to: indexRoute.to })

  expect(router.state.location.pathname).toBe(loginRoute.to)
})

test('the index route sends an unauthorized user to login', async () => {
  serveWorkspaces('unauthorized')

  await router.navigate({ to: indexRoute.to })

  expect(router.state.location.pathname).toBe(loginRoute.to)
})

test('an unknown workspace slug falls back to the first workspace', async () => {
  serveWorkspaces(['acme'])

  await router.navigate({ to: workspaceRoute.to, params: { slug: 'nope' } })

  expect(router.state.location.pathname).toBe(workspaceRoute.fullPath.replace('$slug', 'acme'))
})

test('an unknown workspace slug with no workspaces goes to login', async () => {
  serveWorkspaces([])

  await router.navigate({ to: workspaceRoute.to, params: { slug: 'nope' } })

  expect(router.state.location.pathname).toBe(loginRoute.to)
})

test('the OAuth consent screen sends a signed-out user to login and remembers where to return', async () => {
  serveWorkspaces('unauthorized')

  await router.navigate({
    to: oauthConsentRoute.to,
    search: {
      client_id: 'c',
      redirect_uri: 'https://x.test/cb',
      code_challenge: 'p',
      state: 's',
      scope: 'a',
    },
  })

  expect(router.state.location.pathname).toBe(loginRoute.to)
  expect(router.state.location.search).toMatchObject({
    redirect: expect.stringContaining(oauthConsentRoute.to),
  })
})
