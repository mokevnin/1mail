import { expect, test } from 'vitest'

import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { OverviewPage } from './overview.tsx'

const overview = {
  contacts: { total: 42, active: 40, unsubscribed: 2, newInRange: 5 },
  email: {
    sentCount: 100,
    openedCount: 60,
    clickedCount: 20,
    openRate: 0.6,
    clickRate: 0.2,
    clickToOpenRate: 0.33,
  },
  automations: { total: 3, active: 2, runsActive: 1, runsCompleted: 9 },
  timeseries: [],
}

test('shows the workspace name and contacts count', async () => {
  mockClientFetch((input) => {
    const url = input instanceof Request ? input.url : String(input)
    if (url.includes('/analytics/overview')) {
      return jsonResponse(overview)
    }
    if (url.includes('/workspaces')) {
      return jsonResponse([
        {
          id: '1',
          name: 'Acme',
          slug: 'test',
          collectKey: 'k',
          ingestKey: 'i',
          postalAddress: '',
          createdAt: '2026-01-01T00:00:00Z',
        },
      ])
    }
    return jsonResponse({})
  })

  const { screen } = await renderWithRouter(<OverviewPage />, {
    path: '/workspaces/$slug/',
    initialPath: '/workspaces/test/',
  })

  await expect.element(screen.getByText('Acme')).toBeInTheDocument()
  await expect.element(screen.getByText('42', { exact: true })).toBeInTheDocument()
})
