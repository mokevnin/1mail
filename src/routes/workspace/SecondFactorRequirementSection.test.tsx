import { expect, test } from 'vitest'

import type {
  SiteMembershipResource,
  SiteMembershipRole,
  SiteMembershipsListData,
  SiteUserGetMeData,
  SiteWorkspaceResource,
  SiteWorkspacesListData,
  SiteWorkspacesSetSecondFactorRequirementData,
} from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { formatDate } from '../../utils/datetime.ts'
import { SecondFactorRequirementSection } from './SecondFactorRequirementSection.tsx'

const SLUG = { slug: 'acme' }
const NOW = '2026-01-01T00:00:00Z'

const workspace = (over: Partial<SiteWorkspaceResource> = {}): SiteWorkspaceResource => ({
  id: '1',
  name: 'Acme',
  slug: 'acme',
  collectKey: 'ck',
  ingestKey: 'ik',
  postalAddress: '',
  createdAt: NOW,
  ...over,
})

function roleRoutes(role: SiteMembershipRole) {
  const me = { id: '5', name: 'Me', email: 'me@example.com', emailVerified: true, createdAt: NOW }
  const mine: SiteMembershipResource = {
    id: 'm5',
    userId: '5',
    email: 'me@example.com',
    name: 'Me',
    role,
    createdAt: NOW,
  }
  return [
    route<SiteUserGetMeData>('GET', '/me', {}, () => jsonResponse(me)),
    route<SiteMembershipsListData>('GET', '/workspaces/{slug}/memberships', SLUG, () =>
      jsonResponse([mine]),
    ),
    route<SiteWorkspacesListData>('GET', '/workspaces', {}, () => jsonResponse([workspace()])),
  ]
}

test('an owner switches the requirement on', async () => {
  const bodies: string[] = []
  mockClientRoutes([
    ...roleRoutes('owner'),
    route<SiteWorkspacesSetSecondFactorRequirementData>(
      'PUT',
      '/workspaces/{slug}/second-factor-requirement',
      SLUG,
      async (req) => {
        bodies.push(await req.text())
        return jsonResponse(workspace({ secondFactorRequiredAt: '2026-03-01T00:00:00Z' }))
      },
    ),
  ])
  const { screen } = await renderWithRouter(
    <SecondFactorRequirementSection workspace={workspace()} />,
  )

  const toggle = screen.getByRole('switch', { name: 'Require two-factor authentication' })
  await expect.element(toggle).toBeEnabled()
  await toggle.click()

  await expect.poll(() => bodies).toEqual(['{"required":true}'])
})

test('a member sees the requirement and its grace but cannot change it', async () => {
  mockClientRoutes(roleRoutes('member'))
  const required = workspace({ secondFactorRequiredAt: '2026-03-01T00:00:00Z' })
  const { screen } = await renderWithRouter(<SecondFactorRequirementSection workspace={required} />)

  const toggle = screen.getByRole('switch', { name: 'Require two-factor authentication' })
  await expect.element(toggle).toBeChecked()
  await expect.element(toggle).toBeDisabled()
  await expect
    .element(screen.getByText(new RegExp(`have until ${formatDate('2026-03-08T00:00:00Z')}`)))
    .toBeInTheDocument()
})
