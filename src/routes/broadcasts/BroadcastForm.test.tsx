import { expect, test } from 'vitest'

import type {
  SiteBroadcastsCreateData,
  SiteSegmentsListData,
  SiteTemplatesListData,
} from '../../generated/site/types.gen.ts'
import { broadcastsCreateRoute, broadcastsRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { BroadcastCreatePage } from './resource.tsx'

const SLUG = { slug: 'test' }
const mount = routeMount(broadcastsCreateRoute, { slug: 'test' })

const template = (id: string, subject: string) => ({
  id,
  name: `Template ${id}`,
  subject,
  body: `<mjml>${id}</mjml>`,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
})

function serve(templates: ReturnType<typeof template>[], bodies: unknown[] = []) {
  mockClientRoutes([
    route<SiteSegmentsListData>('GET', '/workspaces/{slug}/segments', SLUG, () =>
      jsonResponse({ items: [{ id: '9', name: 'VIPs' }], totalItems: 1 }),
    ),
    route<SiteTemplatesListData>('GET', '/workspaces/{slug}/templates', SLUG, () =>
      jsonResponse({ items: templates, totalItems: templates.length }),
    ),
    route<SiteBroadcastsCreateData>('POST', '/workspaces/{slug}/broadcasts', SLUG, async (req) => {
      bodies.push(await req.json())
      return jsonResponse({ id: '42', name: 'x' }, { status: 201 })
    }),
  ])
}

test('picking a template copies its subject and body into the form', async () => {
  serve([template('1', 'Welcome aboard')])
  const { screen } = await renderWithRouter(<BroadcastCreatePage />, mount)

  await screen.getByRole('combobox', { name: 'Start from template' }).click()
  await screen.getByRole('option', { name: 'Template 1' }).click()

  await expect.element(screen.getByLabelText(/^Subject/)).toHaveValue('Welcome aboard')
  await expect.element(screen.getByLabelText(/^Email body/)).toHaveValue('<mjml>1</mjml>')
})

test('a template without a subject keeps the subject already typed', async () => {
  serve([template('2', '')])
  const { screen } = await renderWithRouter(<BroadcastCreatePage />, mount)

  await screen.getByLabelText(/^Subject/).fill('My subject')
  await screen.getByRole('combobox', { name: 'Start from template' }).click()
  await screen.getByRole('option', { name: 'Template 2' }).click()

  await expect.element(screen.getByLabelText(/^Email body/)).toHaveValue('<mjml>2</mjml>')
  await expect.element(screen.getByLabelText(/^Subject/)).toHaveValue('My subject')
})

test('the template picker is hidden when the workspace has no templates', async () => {
  serve([])
  const { screen } = await renderWithRouter(<BroadcastCreatePage />, mount)

  await expect.element(screen.getByLabelText(/^Subject/)).toBeInTheDocument()
  expect(screen.getByText('Start from template').elements()).toHaveLength(0)
})

test('Cancel returns to the broadcasts list of the current workspace', async () => {
  serve([])
  const { screen, navigate } = await renderWithRouter(<BroadcastCreatePage />, mount)

  await screen.getByRole('button', { name: 'Cancel' }).click()

  expect(navigate).toHaveBeenCalledWith({ to: broadcastsRoute.to, params: { slug: 'test' } })
})

test('outside a workspace route Cancel does nothing', async () => {
  serve([])
  const { screen, navigate } = await renderWithRouter(<BroadcastCreatePage />, {
    path: '/',
    initialPath: '/',
  })

  await screen.getByRole('button', { name: 'Cancel' }).click()

  expect(navigate).not.toHaveBeenCalled()
})
