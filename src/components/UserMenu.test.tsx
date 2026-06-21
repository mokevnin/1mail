import { afterEach, expect, test, vi } from 'vitest'
import { renderWithRouter } from '../test/renderWithRouter.tsx'
import { UserMenu } from './UserMenu.tsx'

afterEach(() => {
  vi.restoreAllMocks()
})

test('logout calls /auth/logout and navigates to login', async () => {
  const fetchSpy = vi
    .spyOn(globalThis, 'fetch')
    .mockResolvedValue(new Response(null, { status: 200 }))
  const { screen, navigate } = await renderWithRouter(<UserMenu slug="acme" />)

  await screen.getByRole('button').click()
  await screen.getByText('Sign out').click()

  await expect
    .poll(() => fetchSpy.mock.calls.map((c) => String(c[0])))
    .toContainEqual('/auth/logout')
  await expect.poll(() => navigate.mock.calls).toContainEqual([{ to: '/login' }])
})

test('shows workspace settings only when a workspace is active', async () => {
  const { screen } = await renderWithRouter(<UserMenu />)
  await screen.getByRole('button').click()

  await expect.element(screen.getByText('Profile')).toBeInTheDocument()
  expect(screen.container.textContent).not.toContain('Workspace settings')
})
