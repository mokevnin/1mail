import { expect, test } from 'vitest'

import type { SiteAuthRegisterData } from '../../generated/site/types.gen.ts'
import { indexRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { RegisterPage } from './register.tsx'

// route() wants a `path` record; these operations have none, so give it an empty one.
const register = (respond: (req: Request) => Response | Promise<Response>) =>
  route<SiteAuthRegisterData>('POST', '/auth/register', {}, respond)

async function fillAndSubmit(screen: Awaited<ReturnType<typeof renderWithRouter>>['screen']) {
  await screen.getByLabelText(/^Name/).fill('  Ada Lovelace ')
  await screen.getByLabelText(/^Email/).fill(' ada@example.com ')
  await screen.getByLabelText(/^Password/).fill('s3cret-pass')
  await screen.getByRole('button', { name: 'Register' }).click()
}

test('registers with trimmed fields, notifies and navigates home', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    register(async (req) => {
      bodies.push(await req.json())
      return jsonResponse({})
    }),
  ])
  const { screen, navigate } = await renderWithRouter(<RegisterPage />)

  await fillAndSubmit(screen)

  await expect.element(screen.getByText('Account created successfully')).toBeInTheDocument()
  expect(bodies).toEqual([
    { name: 'Ada Lovelace', email: 'ada@example.com', password: 's3cret-pass' },
  ])
  await expect.poll(() => navigate.mock.calls).toContainEqual([{ to: indexRoute.to }])
})

test('shows the API error and clears the password when registration fails', async () => {
  mockClientRoutes([
    register(() => jsonResponse({ detail: 'Email already taken' }, { status: 409 })),
  ])
  const { screen, navigate } = await renderWithRouter(<RegisterPage />)

  await fillAndSubmit(screen)

  await expect.element(screen.getByText('Email already taken')).toBeInTheDocument()
  await expect.element(screen.getByText('Registration failed')).toBeInTheDocument()
  await expect.element(screen.getByLabelText(/^Password/)).toHaveValue('')
  expect(navigate).not.toHaveBeenCalled()
})

test('falls back to the generic message when the error has no detail', async () => {
  mockClientRoutes([register(() => jsonResponse({}, { status: 500 }))])
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
