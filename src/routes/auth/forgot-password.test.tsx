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

test('shows the localized rate limit message on a 429', async () => {
  mockClientFetch(() => jsonResponse({ status: 429, retryAfter: 3600 }, { status: 429 }))
  const { screen } = await renderWithRouter(<ForgotPasswordPage />)

  await screen.getByLabelText(/^Email/).fill('user@example.com')
  await screen.getByRole('button', { name: 'Send reset link' }).click()

  await expect.element(screen.getByText(/Too many requests/)).toBeInTheDocument()
})
