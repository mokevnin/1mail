import { expect, test } from 'vitest'

import {
  activityRoute,
  automationsRoute,
  broadcastsRoute,
  contactsRoute,
  overviewRoute,
  segmentsRoute,
  settingsRoute,
  templatesRoute,
  transactionalEmailsRoute,
} from '../router.tsx'
import { renderWithRouter } from '../test/renderWithRouter.tsx'
import { AppNavbar } from './AppNavbar.tsx'

const SECTIONS = [
  ['Overview', overviewRoute],
  ['Contacts', contactsRoute],
  ['Segments', segmentsRoute],
  ['Broadcasts', broadcastsRoute],
  ['Templates', templatesRoute],
  ['Automations', automationsRoute],
  ['Transactional emails', transactionalEmailsRoute],
  ['Activity', activityRoute],
  ['Settings', settingsRoute],
] as const

test.each(SECTIONS)('%s navigates to its workspace-scoped route', async (label, target) => {
  const { screen, navigate } = await renderWithRouter(<AppNavbar slug="acme" />)

  await screen.getByText(label, { exact: true }).click()

  expect(navigate).toHaveBeenCalledWith({ to: target.to, params: { slug: 'acme' } })
})
