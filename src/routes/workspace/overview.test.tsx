import { expect, test } from 'vitest'

import type {
  SiteAnalyticsOverviewData,
  SiteWorkspacesListData,
} from '../../generated/site/types.gen.ts'
import { overviewRoute } from '../../router.tsx'
import { jsonResponse, mockClientFetch, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
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

const SLUG = { slug: 'test' }
const mount = routeMount(overviewRoute, SLUG)
const workspaces = route<SiteWorkspacesListData>('GET', '/workspaces', {}, () => jsonResponse([]))

const overviewRoutes = (respond: (req: Request) => Response | Promise<Response>) => [
  workspaces,
  route<SiteAnalyticsOverviewData>('GET', '/workspaces/{slug}/analytics/overview', SLUG, respond),
]

test('switching the range refetches the analytics for that window', async () => {
  const ranges: (string | null)[] = []
  mockClientRoutes(
    overviewRoutes((req) => {
      ranges.push(new URL(req.url).searchParams.get('range'))
      return jsonResponse(overview)
    }),
  )
  const { screen } = await renderWithRouter(<OverviewPage />, mount)

  await expect.poll(() => ranges).toEqual(['30d'])
  await screen.getByText('Last 7 days').click()

  await expect.poll(() => ranges).toEqual(['30d', '7d'])
})

test('shows an error alert when analytics fail to load', async () => {
  mockClientRoutes(
    overviewRoutes(() => jsonResponse({ status: 500, detail: 'boom' }, { status: 500 })),
  )
  const { screen } = await renderWithRouter(<OverviewPage />, mount)

  await expect.element(screen.getByText('Failed to load analytics').first()).toBeInTheDocument()
  await expect.element(screen.getByText('boom')).toBeInTheDocument()
})

test('a workspace without contacts shows the empty hint instead of the chart', async () => {
  mockClientRoutes(
    overviewRoutes(() =>
      jsonResponse({
        ...overview,
        contacts: { total: 0, active: 0, unsubscribed: 0, newInRange: 0 },
      }),
    ),
  )
  const { screen } = await renderWithRouter(<OverviewPage />, mount)

  await expect.element(screen.getByText('No contacts yet')).toBeInTheDocument()
})
