import { expect, test } from 'vitest'

import type {
  SiteInvitationResource,
  SiteInvitationsCreateData,
  SiteInvitationsDeleteData,
  SiteInvitationsListData,
  SiteMembershipResource,
  SiteMembershipsDeleteData,
  SiteMembershipsListData,
  SiteMembershipsUpdateData,
} from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { MembersSection } from './MembersSection.tsx'

const SLUG = 'test'

const member: SiteMembershipResource = {
  id: '1',
  userId: '10',
  email: 'ann@example.com',
  name: 'Ann',
  role: 'member',
  createdAt: '2026-01-01T00:00:00Z',
}

const invitation: SiteInvitationResource = {
  id: '5',
  email: 'new@example.com',
  role: 'admin',
  expiresAt: '2026-02-01T00:00:00Z',
  createdAt: '2026-01-01T00:00:00Z',
}

const members = route<SiteMembershipsListData>(
  'GET',
  '/workspaces/{slug}/memberships',
  { slug: SLUG },
  () => jsonResponse([member]),
)
const invites = (items: SiteInvitationResource[]) =>
  route<SiteInvitationsListData>('GET', '/workspaces/{slug}/invitations', { slug: SLUG }, () =>
    jsonResponse(items),
  )

test('lists members and pending invitations', async () => {
  mockClientRoutes([members, invites([invitation])])
  const { screen } = await renderWithRouter(<MembersSection slug={SLUG} />)

  await expect.element(screen.getByText('ann@example.com')).toBeInTheDocument()
  await expect.element(screen.getByText('new@example.com')).toBeInTheDocument()
})

test('invites a member and reveals the invite link', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    members,
    invites([]),
    route<SiteInvitationsCreateData>(
      'POST',
      '/workspaces/{slug}/invitations',
      { slug: SLUG },
      async (req) => {
        bodies.push(await req.json())
        return jsonResponse(
          { inviteUrl: 'https://app.test/invite/abc', resource: invitation },
          { status: 201 },
        )
      },
    ),
  ])
  const { screen } = await renderWithRouter(<MembersSection slug={SLUG} />)

  await screen
    .getByLabelText(/^Email/)
    .last()
    .fill(' new@example.com ')
  await screen.getByRole('button', { name: 'Invite' }).click()

  await expect.element(screen.getByText('https://app.test/invite/abc')).toBeInTheDocument()
  expect(bodies).toEqual([{ email: 'new@example.com', role: 'member' }])

  // The notice can be dismissed.
  await screen.getByRole('button', { name: 'Copy' }).click()
  await screen.getByRole('alert').getByRole('button').last().click()
  await expect.element(screen.getByText('https://app.test/invite/abc')).not.toBeInTheDocument()
})

test('revokes an invitation after confirmation', async () => {
  let revoked = false
  mockClientRoutes([
    members,
    invites([invitation]),
    route<SiteInvitationsDeleteData>(
      'DELETE',
      '/workspaces/{slug}/invitations/{id}',
      { slug: SLUG, id: '5' },
      () => {
        revoked = true
        return new Response(null, { status: 204 })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<MembersSection slug={SLUG} />)

  await screen.getByRole('button', { name: 'Revoke' }).click()
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => revoked).toBe(true)
})

test('removes a member after confirmation', async () => {
  let removed = false
  mockClientRoutes([
    members,
    invites([]),
    route<SiteMembershipsDeleteData>(
      'DELETE',
      '/workspaces/{slug}/memberships/{id}',
      { slug: SLUG, id: '1' },
      () => {
        removed = true
        return new Response(null, { status: 204 })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<MembersSection slug={SLUG} />)

  await screen.getByRole('button', { name: 'Remove' }).click()
  await screen.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.poll(() => removed).toBe(true)
})

test('changes a member role', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    members,
    invites([]),
    route<SiteMembershipsUpdateData>(
      'PUT',
      '/workspaces/{slug}/memberships/{id}',
      { slug: SLUG, id: '1' },
      async (req) => {
        bodies.push(await req.json())
        return jsonResponse({ ...member, role: 'admin' })
      },
    ),
  ])
  const { screen } = await renderWithRouter(<MembersSection slug={SLUG} />)

  await screen.getByRole('combobox').first().click()
  await screen.getByRole('option', { name: 'admin' }).first().click()

  await expect.poll(() => bodies).toEqual([{ role: 'admin' }])
})

test('shows an error alert when members fail to load', async () => {
  mockClientRoutes([
    route<SiteMembershipsListData>('GET', '/workspaces/{slug}/memberships', { slug: SLUG }, () =>
      jsonResponse({ status: 500, detail: 'boom' }, { status: 500 }),
    ),
    invites([]),
  ])
  const { screen } = await renderWithRouter(<MembersSection slug={SLUG} />)

  await expect.element(screen.getByText('Failed to load members').first()).toBeInTheDocument()
})
