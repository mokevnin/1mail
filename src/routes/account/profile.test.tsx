import { expect, test } from 'vitest'
import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { ProfilePage } from './profile.tsx'

const user = { id: '1', name: 'John', email: 'info@1mail.com', createdAt: '2026-01-01T00:00:00Z' }

test('loads the profile and submits a name change', async () => {
  const puts: string[] = []
  mockClientFetch(async (input) => {
    const req = input instanceof Request ? input : new Request(String(input))
    if (req.method === 'PUT') {
      puts.push(await req.clone().text())
      return jsonResponse({ ...user, name: 'Renamed' })
    }
    return jsonResponse(user)
  })

  const { screen } = await renderWithRouter(<ProfilePage />)

  const nameInput = screen.getByLabelText('Name')
  await expect.element(nameInput).toHaveValue('John')

  await nameInput.fill('Renamed')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.poll(() => puts.length).toBeGreaterThan(0)
  expect(puts[0]).toContain('"name":"Renamed"')
})

test('email is read-only', async () => {
  mockClientFetch(() => jsonResponse(user))
  const { screen } = await renderWithRouter(<ProfilePage />)

  await expect.element(screen.getByLabelText('Email')).toBeDisabled()
})
