import { expect, test } from 'vitest'

import type {
  SiteAutomationsGetData,
  SiteAutomationsUpdateData,
} from '../../generated/site/types.gen.ts'
import { automationsEditRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { AutomationEditPage } from './edit.tsx'

const PATH = { slug: 'test', id: '5' }
const EDIT_ROUTE = routeMount(automationsEditRoute, { slug: 'test', automationId: '5' })
const NOW = '2026-01-01T00:00:00Z'

function getRoute(respond: () => Response) {
  return route<SiteAutomationsGetData>('GET', '/workspaces/{slug}/automations/{id}', PATH, respond)
}

test('loads the automation into the builder', async () => {
  mockClientRoutes([
    getRoute(() =>
      jsonResponse({
        id: '5',
        name: 'Welcome flow',
        status: 'draft',
        triggerEvent: 'email.clicked',
        steps: [{ type: 'wait', seconds: 60 }],
        createdAt: NOW,
        updatedAt: NOW,
      }),
    ),
  ])

  const { screen } = await renderWithRouter(<AutomationEditPage />, EDIT_ROUTE)

  await expect.element(screen.getByText('Edit automation')).toBeInTheDocument()
  await expect.element(screen.getByLabelText('Name', { exact: false })).toHaveValue('Welcome flow')
  await expect
    .element(screen.getByRole('combobox', { name: 'Trigger event' }))
    .toHaveValue('Email clicked')
})

test('shows an error alert when the automation fails to load', async () => {
  mockClientRoutes([getRoute(() => jsonResponse({ title: 'Gone', status: 404 }, { status: 404 }))])

  const { screen } = await renderWithRouter(<AutomationEditPage />, EDIT_ROUTE)

  await expect.element(screen.getByText('Failed to load automations').first()).toBeInTheDocument()
})

const STORED = {
  id: '5',
  name: 'Welcome flow',
  status: 'draft',
  triggerEvent: 'email.clicked',
  steps: [
    { type: 'wait', seconds: 60 },
    { type: 'apply_tag', tag: 'vip' },
  ],
  createdAt: NOW,
  updatedAt: NOW,
}

function updateRoute(respond: (req: Request) => Response | Promise<Response>) {
  return route<SiteAutomationsUpdateData>(
    'PUT',
    '/workspaces/{slug}/automations/{id}',
    PATH,
    respond,
  )
}

test('saving from the builder sends the edited name, trigger and the canvas steps', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    getRoute(() => jsonResponse(STORED)),
    updateRoute(async (req) => {
      bodies.push(await req.json())
      return jsonResponse(STORED)
    }),
  ])

  const { screen } = await renderWithRouter(<AutomationEditPage />, EDIT_ROUTE)
  await screen.getByLabelText('Name', { exact: false }).fill('  Renamed  ')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect
    .poll(() => bodies)
    .toEqual([
      {
        name: 'Renamed',
        triggerEvent: 'email.clicked',
        steps: [
          { type: 'wait', seconds: 60 },
          { type: 'apply_tag', tag: 'vip' },
        ],
      },
    ])
  await expect.element(screen.getByText('Automation updated')).toBeInTheDocument()
})

test('a failed save from the builder shows an error toast', async () => {
  mockClientRoutes([
    getRoute(() => jsonResponse(STORED)),
    updateRoute(() => jsonResponse({ title: 'Boom', status: 500 }, { status: 500 })),
  ])

  const { screen } = await renderWithRouter(<AutomationEditPage />, EDIT_ROUTE)
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.element(screen.getByText('Failed to save automation')).toBeInTheDocument()
})
