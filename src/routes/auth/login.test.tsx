import { expect, test } from 'vitest'

import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { LoginPage } from './login.tsx'

test('navigates home after a successful login', async () => {
  mockClientFetch(() => jsonResponse({}))
  const { screen, navigate } = await renderWithRouter(<LoginPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  // Required fields render their label as "Password *", hence the anchored regex.
  await screen.getByLabelText(/^Password/).fill('secret')
  await screen.getByRole('button', { name: 'Sign in' }).click()

  await expect.poll(() => navigate.mock.calls).toContainEqual([{ to: '/' }])
})

test('shows an error notification when login fails', async () => {
  mockClientFetch(() => jsonResponse({ detail: 'Invalid credentials' }, { status: 401 }))
  const { screen, navigate } = await renderWithRouter(<LoginPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  // Required fields render their label as "Password *", hence the anchored regex.
  await screen.getByLabelText(/^Password/).fill('wrong')
  await screen.getByRole('button', { name: 'Sign in' }).click()

  await expect.element(screen.getByText('Invalid credentials')).toBeInTheDocument()
  expect(navigate).not.toHaveBeenCalled()
})

test('shows how long to wait when login is rate limited', async () => {
  mockClientFetch(() => jsonResponse({ status: 429, retryAfter: 90 }, { status: 429 }))
  const { screen } = await renderWithRouter(<LoginPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  await screen.getByLabelText(/^Password/).fill('secret')
  await screen.getByRole('button', { name: 'Sign in' }).click()

  await expect.element(screen.getByText(/Try again in 2 minutes/)).toBeInTheDocument()
})

test('shows the wait in seconds when it is under a minute', async () => {
  mockClientFetch(() => jsonResponse({ status: 429, retryAfter: 4 }, { status: 429 }))
  const { screen } = await renderWithRouter(<LoginPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  await screen.getByLabelText(/^Password/).fill('secret')
  await screen.getByRole('button', { name: 'Sign in' }).click()

  await expect.element(screen.getByText(/Try again in 4 seconds/)).toBeInTheDocument()
})
