import { expect, test } from 'vitest'
import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { OverviewPage } from './overview.tsx'

test('shows the workspace name and contacts count', async () => {
  mockClientFetch((input) => {
    const url = input instanceof Request ? input.url : String(input)
    if (url.includes('/contacts')) {
      return jsonResponse({ items: [], page: 1, pageSize: 1, totalItems: 7, totalPages: 7 })
    }
    if (url.includes('/workspaces')) {
      return jsonResponse([
        { id: '1', name: 'Acme', slug: 'test', collectKey: 'k', createdAt: '2026-01-01T00:00:00Z' },
      ])
    }
    return jsonResponse({})
  })

  const { screen } = await renderWithRouter(<OverviewPage />)

  await expect.element(screen.getByText('Acme')).toBeInTheDocument()
  await expect.element(screen.getByText('7')).toBeInTheDocument()
})
