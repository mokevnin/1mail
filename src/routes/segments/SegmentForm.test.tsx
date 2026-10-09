import { expect, test } from 'vitest'

import type {
  SiteCustomFieldsListData,
  SiteEventsActionsData,
} from '../../generated/site/types.gen.ts'
import { segmentsCreateRoute, segmentsRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { SegmentCreatePage } from './resource.tsx'

const SLUG = { slug: 'test' }

const catalogue = [
  route<SiteEventsActionsData>('GET', '/workspaces/{slug}/events/actions', SLUG, () =>
    jsonResponse({ actions: [] }),
  ),
  route<SiteCustomFieldsListData>('GET', '/workspaces/{slug}/custom-fields', SLUG, () =>
    jsonResponse({ items: [], totalItems: 0 }),
  ),
]

test('Cancel returns to the segments list of the current workspace', async () => {
  mockClientRoutes(catalogue)
  const { screen, navigate } = await renderWithRouter(
    <SegmentCreatePage />,
    routeMount(segmentsCreateRoute, { slug: 'test' }),
  )

  await screen.getByRole('button', { name: 'Cancel' }).click()

  expect(navigate).toHaveBeenCalledWith({ to: segmentsRoute.to, params: { slug: 'test' } })
})

test('outside a workspace route there is no rule builder and Cancel does nothing', async () => {
  mockClientRoutes(catalogue)
  const { screen, navigate } = await renderWithRouter(<SegmentCreatePage />, {
    path: '/',
    initialPath: '/',
  })

  await screen.getByRole('button', { name: 'Cancel' }).click()

  expect(navigate).not.toHaveBeenCalled()
  expect(screen.getByRole('button', { name: 'Preview' }).elements()).toHaveLength(0)
})
