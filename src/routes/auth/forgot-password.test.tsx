import { expect, test } from 'vitest'

import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { ForgotPasswordPage } from './forgot-password.tsx'

test('shows the confirmation state after submitting', async () => {
  // The API returns 202 whether or not the address exists.
  mockClientFetch(() => jsonResponse({}, { status: 202 }))
  const { screen } = await renderWithRouter(<ForgotPasswordPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  await screen.getByRole('button', { name: 'Send reset link' }).click()

  await expect.element(screen.getByText('Check your email')).toBeInTheDocument()
})

test('tells how many minutes to wait when the 429 carries a retry time', async () => {
  mockClientFetch(() => jsonResponse({ status: 429, retryAfter: 3600 }, { status: 429 }))
  const { screen } = await renderWithRouter(<ForgotPasswordPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  await screen.getByRole('button', { name: 'Send reset link' }).click()

  await expect.element(screen.getByText(/Try again in 60 minutes/)).toBeInTheDocument()
})

test('tells how many seconds to wait below a minute', async () => {
  mockClientFetch(() => jsonResponse({ status: 429, retryAfter: 45 }, { status: 429 }))
  const { screen } = await renderWithRouter(<ForgotPasswordPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  await screen.getByRole('button', { name: 'Send reset link' }).click()

  await expect.element(screen.getByText(/Try again in 45 seconds/)).toBeInTheDocument()
})

test('falls back to the generic message when the 429 carries no retry time', async () => {
  mockClientFetch(() => jsonResponse({ status: 429 }, { status: 429 }))
  const { screen } = await renderWithRouter(<ForgotPasswordPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  await screen.getByRole('button', { name: 'Send reset link' }).click()

  await expect.element(screen.getByText(/Too many requests/)).toBeInTheDocument()
})
