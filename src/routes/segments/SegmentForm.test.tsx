import { expect, test } from 'vitest'

import {
  handleSiteCustomFieldsList,
  handleSiteEventsActions,
} from '../../generated/site/msw.gen.ts'
import { segmentsCreateRoute, segmentsRoute } from '../../router.tsx'
import { page } from '../../test/payloads.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { worker } from '../../test/worker.ts'
import { SegmentCreatePage } from './resource.tsx'

function serveCatalogue() {
  worker.use(
    handleSiteEventsActions({ body: { actions: [] } }),
    handleSiteCustomFieldsList({ body: page([], 0) }),
  )
}

test('Cancel returns to the segments list of the current workspace', async () => {
  serveCatalogue()
  const { screen, navigate } = await renderWithRouter(
    <SegmentCreatePage />,
    routeMount(segmentsCreateRoute, { slug: 'test' }),
  )

  await screen.getByRole('button', { name: 'Cancel' }).click()

  expect(navigate).toHaveBeenCalledWith({ to: segmentsRoute.to, params: { slug: 'test' } })
})

test('outside a workspace route there is no rule builder and Cancel does nothing', async () => {
  serveCatalogue()
  const { screen, navigate } = await renderWithRouter(<SegmentCreatePage />, {
    path: '/',
    initialPath: '/',
  })

  await screen.getByRole('button', { name: 'Cancel' }).click()

  expect(navigate).not.toHaveBeenCalled()
  expect(screen.getByRole('button', { name: 'Preview' }).elements()).toHaveLength(0)
})
