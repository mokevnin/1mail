import { expect, test } from 'vitest'
import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { LoginPage } from './login.tsx'

test('navigates home after a successful login', async () => {
  mockClientFetch(() => jsonResponse({}))
  const { screen, navigate } = await renderWithRouter(<LoginPage />)

  await screen.getByLabelText('Email').fill('user@example.com')
  // Mantine's PasswordInput renders the input plus a "Toggle password
  // visibility" button whose accessible name also matches "Password"; the input
  // is first in DOM order, so .first() targets it.
  await screen.getByLabelText('Password').first().fill('secret')
  await screen.getByRole('button', { name: 'Sign in' }).click()

  await expect.poll(() => navigate.mock.calls).toContainEqual([{ to: '/' }])
})

test('shows an error notification when login fails', async () => {
  mockClientFetch(() => jsonResponse({ detail: 'Invalid credentials' }, { status: 401 }))
  const { screen, navigate } = await renderWithRouter(<LoginPage />)

  await screen.getByLabelText('Email').fill('user@example.com')
  // Mantine's PasswordInput renders the input plus a "Toggle password
  // visibility" button whose accessible name also matches "Password"; the input
  // is first in DOM order, so .first() targets it.
  await screen.getByLabelText('Password').first().fill('wrong')
  await screen.getByRole('button', { name: 'Sign in' }).click()

  await expect.element(screen.getByText('Invalid credentials')).toBeInTheDocument()
  expect(navigate).not.toHaveBeenCalled()
})
