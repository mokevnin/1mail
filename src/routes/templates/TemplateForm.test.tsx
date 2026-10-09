import { expect, test } from 'vitest'

import { templatesCreateRoute, templatesRoute } from '../../router.tsx'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { TemplateCreatePage } from './resource.tsx'

test('Cancel returns to the templates list of the current workspace', async () => {
  const { screen, navigate } = await renderWithRouter(
    <TemplateCreatePage />,
    routeMount(templatesCreateRoute, { slug: 'test' }),
  )

  await screen.getByRole('button', { name: 'Cancel' }).click()

  expect(navigate).toHaveBeenCalledWith({ to: templatesRoute.to, params: { slug: 'test' } })
})

test('Cancel does nothing outside a workspace route', async () => {
  const { screen, navigate } = await renderWithRouter(<TemplateCreatePage />, {
    path: '/',
    initialPath: '/',
  })

  await screen.getByRole('button', { name: 'Cancel' }).click()

  expect(navigate).not.toHaveBeenCalled()
})
