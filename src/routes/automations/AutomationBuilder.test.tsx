import { expect, test, vi } from 'vitest'

import type {
  SiteAutomationsGetData,
  SiteAutomationsUpdateData,
} from '../../generated/site/types.gen.ts'
import { automationsEditRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { AutomationEditPage } from './edit.tsx'

// The canvas can only drop branches when a node has several outgoing edges, which is
// impractical to draw in a test; report one dropped branch from the real conversion instead.
vi.mock(import('./definition.ts'), async (importOriginal) => {
  const original = await importOriginal()
  return {
    ...original,
    graphToSteps: (graph) => ({ ...original.graphToSteps(graph), dropped: 1 }),
  }
})

const PATH = { slug: 'test', id: '5' }
const EDIT_ROUTE = routeMount(automationsEditRoute, { slug: 'test', automationId: '5' })
const NOW = '2026-01-01T00:00:00Z'
const STORED = {
  id: '5',
  name: 'Welcome flow',
  status: 'draft',
  triggerEvent: 'email.clicked',
  steps: [{ type: 'wait', seconds: 60 }],
  createdAt: NOW,
  updatedAt: NOW,
}

test('saving a graph whose branches cannot be represented warns that they were dropped', async () => {
  mockClientRoutes([
    route<SiteAutomationsGetData>('GET', '/workspaces/{slug}/automations/{id}', PATH, () =>
      jsonResponse(STORED),
    ),
    route<SiteAutomationsUpdateData>('PUT', '/workspaces/{slug}/automations/{id}', PATH, () =>
      jsonResponse(STORED),
    ),
  ])

  const { screen } = await renderWithRouter(<AutomationEditPage />, EDIT_ROUTE)
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.element(screen.getByText('Automation updated')).toBeInTheDocument()
  await expect.element(screen.getByText('Branches not supported yet')).toBeInTheDocument()
})
