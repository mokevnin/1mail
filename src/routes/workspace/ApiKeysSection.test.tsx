import { afterEach, expect, test, vi } from 'vitest'

import type {
  SiteTokensCreateData,
  SiteTokensDeleteData,
  SiteTokensListData,
  SiteApiTokenResource,
} from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientFetch, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { ApiKeysSection } from './ApiKeysSection.tsx'

afterEach(() => {
  vi.restoreAllMocks()
})

test('creates a token and reveals the one-time secret', async () => {
  let created = false
  mockClientFetch((input) => {
    const req = input instanceof Request ? input : new Request(String(input))
    if (req.method === 'POST') {
      created = true
      return jsonResponse(
        {
          token: 'omtk_newprefix_supersecret',
          resource: {
            id: '2',
            name: 'CI',
            prefix: 'newprefix',
            scopes: ['contacts:read'],
            createdAt: '2026-06-01T00:00:00Z',
          },
        },
        { status: 201 },
      )
    }
    // GET list — empty before creation, the created token after.
    return jsonResponse(
      created
        ? [
            {
              id: '2',
              name: 'CI',
              prefix: 'newprefix',
              scopes: ['contacts:read'],
              createdAt: '2026-06-01T00:00:00Z',
            },
          ]
        : [],
    )
  })

  const { screen } = await renderWithRouter(<ApiKeysSection slug="test" />)

  await screen.getByLabelText(/^Token name/).fill('CI')
  await screen.getByRole('button', { name: 'Create token' }).click()

  // The full secret is shown once.
  await expect.element(screen.getByText('omtk_newprefix_supersecret')).toBeInTheDocument()
})

test('a plain member who is forbidden to create a token sees why', async () => {
  mockClientFetch((input) => {
    const req = input instanceof Request ? input : new Request(String(input))
    if (req.method === 'POST') {
      return jsonResponse(
        { status: 403, detail: 'only owners and admins can manage API tokens' },
        { status: 403 },
      )
    }
    return jsonResponse([])
  })

  const { screen } = await renderWithRouter(<ApiKeysSection slug="test" />)

  await screen.getByLabelText(/^Token name/).fill('CI')
  await screen.getByRole('button', { name: 'Create token' }).click()

  await expect
    .element(screen.getByText(/Only workspace owners and admins can manage API tokens/))
    .toBeInTheDocument()
})

const SLUG = { slug: 'test' }
const REVOKE_FAILED = 'Failed to revoke token'

const token: SiteApiTokenResource = {
  id: '2',
  name: 'CI',
  prefix: 'newprefix',
  scopes: ['contacts:read'],
  createdAt: '2026-06-01T00:00:00Z',
}

const listTokens = (items: SiteApiTokenResource[]) =>
  route<SiteTokensListData>('GET', '/workspaces/{slug}/tokens', SLUG, () => jsonResponse(items))

const deleteToken = (respond: () => Response) =>
  route<SiteTokensDeleteData>(
    'DELETE',
    '/workspaces/{slug}/tokens/{id}',
    { ...SLUG, id: '2' },
    respond,
  )

test('revokes a token after confirmation', async () => {
  let revoked = false
  mockClientRoutes([
    listTokens([token]),
    deleteToken(() => {
      revoked = true
      return new Response(null, { status: 204 })
    }),
  ])
  const { screen } = await renderWithRouter(<ApiKeysSection slug="test" />)

  await screen.getByRole('button', { name: 'Revoke' }).click()
  await expect.element(screen.getByText('Revoke token?')).toBeInTheDocument()
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => revoked).toBe(true)
})

test('a failed revoke shows the error toast', async () => {
  mockClientRoutes([
    listTokens([token]),
    deleteToken(() => jsonResponse({ status: 500, detail: 'boom' }, { status: 500 })),
  ])
  const { screen } = await renderWithRouter(<ApiKeysSection slug="test" />)

  await screen.getByRole('button', { name: 'Revoke' }).click()
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.element(screen.getByText(REVOKE_FAILED)).toBeInTheDocument()
})

test('shows an error alert when the tokens fail to load', async () => {
  mockClientRoutes([
    route<SiteTokensListData>('GET', '/workspaces/{slug}/tokens', SLUG, () =>
      jsonResponse({ status: 500, detail: 'boom' }, { status: 500 }),
    ),
  ])
  const { screen } = await renderWithRouter(<ApiKeysSection slug="test" />)

  await expect
    .element(screen.getByRole('alert', { name: 'Failed to load tokens' }))
    .toBeInTheDocument()
})

test('the one-time secret can be copied and dismissed', async () => {
  vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue()
  mockClientRoutes([
    listTokens([]),
    route<SiteTokensCreateData>('POST', '/workspaces/{slug}/tokens', SLUG, () =>
      jsonResponse({ token: 'omtk_secret_value', resource: token }, { status: 201 }),
    ),
  ])
  const { screen } = await renderWithRouter(<ApiKeysSection slug="test" />)

  await screen.getByLabelText(/^Token name/).fill('CI')
  await screen.getByRole('button', { name: 'Create token' }).click()
  await expect.element(screen.getByText('omtk_secret_value')).toBeInTheDocument()

  await screen.getByRole('button', { name: 'Copy', exact: true }).click()
  await expect.element(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument()

  await screen
    .getByRole('alert', { name: /Copy this token now/ })
    .getByRole('button')
    .last()
    .click()
  await expect.element(screen.getByText('omtk_secret_value')).not.toBeInTheDocument()
})
