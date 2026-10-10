import { HttpResponse } from 'msw'
import { expect, test } from 'vitest'

import { handleSiteAuthRegister } from '../../generated/site/msw.gen.ts'
import { indexRoute } from '../../router.tsx'
import { problem } from '../../test/problem.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { worker } from '../../test/worker.ts'
import { RegisterPage } from './register.tsx'

async function fillAndSubmit(screen: Awaited<ReturnType<typeof renderWithRouter>>['screen']) {
  await screen.getByLabelText(/^Name/).fill('  Ada Lovelace ')
  await screen.getByLabelText(/^Email/).fill(' ada@example.com ')
  await screen.getByLabelText(/^Password/).fill('s3cret-pass')
  await screen.getByRole('button', { name: 'Register' }).click()
}

test('registers with trimmed fields, notifies and navigates home', async () => {
  const bodies: unknown[] = []
  worker.use(
    handleSiteAuthRegister(async ({ request }) => {
      bodies.push(await request.json())
      return HttpResponse.json({})
    }),
  )
  const { screen, navigate } = await renderWithRouter(<RegisterPage />)

  await fillAndSubmit(screen)

  await expect.element(screen.getByText('Account created successfully')).toBeInTheDocument()
  expect(bodies).toEqual([
    { name: 'Ada Lovelace', email: 'ada@example.com', password: 's3cret-pass' },
  ])
  await expect.poll(() => navigate.mock.calls).toContainEqual([{ to: indexRoute.to }])
})

test('shows the API error and clears the password when registration fails', async () => {
  worker.use(handleSiteAuthRegister(() => problem(409, { detail: 'Email already taken' })))
  const { screen, navigate } = await renderWithRouter(<RegisterPage />)

  await fillAndSubmit(screen)

  await expect.element(screen.getByText('Email already taken')).toBeInTheDocument()
  await expect.element(screen.getByText('Registration failed')).toBeInTheDocument()
  await expect.element(screen.getByLabelText(/^Password/)).toHaveValue('')
  expect(navigate).not.toHaveBeenCalled()
})

test('falls back to the generic message when the error has no detail', async () => {
  worker.use(handleSiteAuthRegister(() => problem(500, {})))
  const { screen } = await renderWithRouter(<RegisterPage />)

  await fillAndSubmit(screen)

  await expect
    .element(
      screen.getByText("We couldn't create your account. Please check your details and try again."),
    )
    .toBeInTheDocument()
})

test('links back to sign in', async () => {
  const { screen } = await renderWithRouter(<RegisterPage />)
  await expect
    .element(screen.getByRole('link', { name: 'Already have an account? Sign in' }))
    .toBeInTheDocument()
})
