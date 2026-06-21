import { expect, test } from 'vitest'
import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { ApiKeysSection } from './ApiKeysSection.tsx'

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

  await screen.getByLabelText('Token name').fill('CI')
  await screen.getByRole('button', { name: 'Create token' }).click()

  // The full secret is shown once.
  await expect.element(screen.getByText('omtk_newprefix_supersecret')).toBeInTheDocument()
})
