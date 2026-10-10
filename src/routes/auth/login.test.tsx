import { HttpResponse } from 'msw'
import { expect, test } from 'vitest'

import { handleSiteAuthDirectLogin } from '../../generated/site/msw.gen.ts'
import { problem } from '../../test/problem.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { worker } from '../../test/worker.ts'
import { LoginPage } from './login.tsx'

test('navigates home after a successful login', async () => {
  worker.use(handleSiteAuthDirectLogin(() => HttpResponse.json({})))
  const { screen, navigate } = await renderWithRouter(<LoginPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  // Required fields render their label as "Password *", hence the anchored regex.
  await screen.getByLabelText(/^Password/).fill('secret')
  await screen.getByRole('button', { name: 'Sign in' }).click()

  await expect.poll(() => navigate.mock.calls).toContainEqual([{ to: '/' }])
})

test('shows an error notification when login fails', async () => {
  worker.use(handleSiteAuthDirectLogin(() => problem(401, { detail: 'Invalid credentials' })))
  const { screen, navigate } = await renderWithRouter(<LoginPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  // Required fields render their label as "Password *", hence the anchored regex.
  await screen.getByLabelText(/^Password/).fill('wrong')
  await screen.getByRole('button', { name: 'Sign in' }).click()

  await expect.element(screen.getByText('Invalid credentials')).toBeInTheDocument()
  expect(navigate).not.toHaveBeenCalled()
})

test('shows how long to wait when login is rate limited', async () => {
  worker.use(handleSiteAuthDirectLogin(() => problem(429, { retryAfter: 90 })))
  const { screen } = await renderWithRouter(<LoginPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  await screen.getByLabelText(/^Password/).fill('secret')
  await screen.getByRole('button', { name: 'Sign in' }).click()

  await expect.element(screen.getByText(/Try again in 2 minutes/)).toBeInTheDocument()
})

test('shows the wait in seconds when it is under a minute', async () => {
  worker.use(handleSiteAuthDirectLogin(() => problem(429, { retryAfter: 4 })))
  const { screen } = await renderWithRouter(<LoginPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  await screen.getByLabelText(/^Password/).fill('secret')
  await screen.getByRole('button', { name: 'Sign in' }).click()

  await expect.element(screen.getByText(/Try again in 4 seconds/)).toBeInTheDocument()
})
