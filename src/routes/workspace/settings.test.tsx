import { expect, test } from 'vitest'
import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { SettingsPage } from './settings.tsx'

const workspace = {
  id: '1',
  name: 'Acme',
  slug: 'test',
  collectKey: 'omck_test_key',
  createdAt: '2026-01-01T00:00:00Z',
}

test('renames the workspace and shows the tracking snippet', async () => {
  const puts: string[] = []
  mockClientFetch(async (input) => {
    const req = input instanceof Request ? input : new Request(String(input))
    if (req.method === 'PUT') {
      puts.push(await req.clone().text())
      return jsonResponse({ ...workspace, name: 'Acme Inc' })
    }
    // The settings page also lists API tokens for its keys section.
    if (req.url.includes('/tokens')) {
      return jsonResponse([])
    }
    return jsonResponse([workspace])
  })

  const { screen } = await renderWithRouter(<SettingsPage />)

  // Tracking snippet includes the collect key.
  await expect.element(screen.getByText(/omck_test_key/)).toBeInTheDocument()

  const nameInput = screen.getByLabelText('Workspace name')
  await expect.element(nameInput).toHaveValue('Acme')
  await nameInput.fill('Acme Inc')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.poll(() => puts.length).toBeGreaterThan(0)
  expect(puts[0]).toContain('"name":"Acme Inc"')
})
