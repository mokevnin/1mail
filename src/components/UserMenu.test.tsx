import { afterEach, expect, test, vi } from 'vitest'

import { profileRoute, settingsRoute } from '../router.tsx'
import { renderWithRouter } from '../test/renderWithRouter.tsx'
import { UserMenu } from './UserMenu.tsx'

afterEach(() => {
  vi.restoreAllMocks()
})

function requestUrl(input: Parameters<typeof fetch>[0]): string {
  if (typeof input === 'string') return input
  if (input instanceof URL) return input.pathname
  return input.url
}

test('logout calls /auth/logout and navigates to login', async () => {
  const fetchSpy = vi
    .spyOn(globalThis, 'fetch')
    .mockResolvedValue(new Response(null, { status: 200 }))
  const { screen, navigate } = await renderWithRouter(<UserMenu slug="acme" />)

  await screen.getByRole('button').click()
  await screen.getByText('Sign out').click()

  await expect
    .poll(() => fetchSpy.mock.calls.map(([input]) => requestUrl(input)))
    .toContainEqual('/auth/logout')
  await expect.poll(() => navigate.mock.calls).toContainEqual([{ to: '/login' }])
})

test('shows workspace settings only when a workspace is active', async () => {
  const { screen } = await renderWithRouter(<UserMenu />)
  await screen.getByRole('button').click()

  await expect.element(screen.getByText('Profile')).toBeInTheDocument()
  expect(screen.container.textContent).not.toContain('Workspace settings')
})

test('profile item navigates to the profile page', async () => {
  const { screen, navigate } = await renderWithRouter(<UserMenu slug="acme" />)
  await screen.getByRole('button').click()
  await screen.getByText('Profile').click()

  await expect.poll(() => navigate.mock.calls).toContainEqual([{ to: profileRoute.to }])
})

test('workspace settings item navigates with the active slug', async () => {
  const { screen, navigate } = await renderWithRouter(<UserMenu slug="acme" />)
  await screen.getByRole('button').click()
  await screen.getByText('Workspace settings').click()

  await expect
    .poll(() => navigate.mock.calls)
    .toContainEqual([{ to: settingsRoute.to, params: { slug: 'acme' } }])
})

test('a failed logout shows an error and stays put', async () => {
  vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('offline'))
  const { screen, navigate } = await renderWithRouter(<UserMenu slug="acme" />)
  await screen.getByRole('button').click()
  await screen.getByText('Sign out').click()

  await expect.element(screen.getByText('Sign out failed').first()).toBeInTheDocument()
  expect(navigate).not.toHaveBeenCalled()
})

test('shows the label and its initial in the avatar when given', async () => {
  const { screen } = await renderWithRouter(<UserMenu label="ada@example.com" />)

  await expect.element(screen.getByText('ada@example.com')).toBeInTheDocument()
  await expect.element(screen.getByText('A', { exact: true })).toBeInTheDocument()
})
