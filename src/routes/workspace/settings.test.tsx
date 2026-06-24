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

function eventsPage(totalItems: number) {
  return jsonResponse({ items: [], page: 1, pageSize: 1, totalItems, totalPages: 0 })
}

test('renames the workspace and shows the tracking snippet and test command', async () => {
  const puts: string[] = []
  mockClientFetch(async (input) => {
    const req = input instanceof Request ? input : new Request(String(input))
    if (req.method === 'PUT') {
      puts.push(await req.clone().text())
      return jsonResponse({ ...workspace, name: 'Acme Inc' })
    }
    // Install status polls the events feed (no events yet).
    if (req.url.includes('/events')) {
      return eventsPage(0)
    }
    // The settings page also lists API tokens for its keys section.
    if (req.url.includes('/tokens')) {
      return jsonResponse([])
    }
    return jsonResponse([workspace])
  })

  const { screen } = await renderWithRouter(<SettingsPage />)

  // Tracking snippet and the curl test command both carry the collect key.
  await expect.element(screen.getByText(/omck_test_key/).first()).toBeInTheDocument()
  await expect.element(screen.getByText(/x-collect-key: omck_test_key/)).toBeInTheDocument()
  // No events yet → waiting state.
  await expect.element(screen.getByText('Waiting for the first event…')).toBeInTheDocument()

  const nameInput = screen.getByLabelText('Workspace name')
  await expect.element(nameInput).toHaveValue('Acme')
  await nameInput.fill('Acme Inc')
  await screen.getByRole('button', { name: 'Save' }).click()

  await expect.poll(() => puts.length).toBeGreaterThan(0)
  expect(puts[0]).toContain('"name":"Acme Inc"')
})

test('shows the connected install status once events arrive', async () => {
  mockClientFetch((input) => {
    const req = input instanceof Request ? input : new Request(String(input))
    if (req.url.includes('/events')) {
      return eventsPage(7)
    }
    if (req.url.includes('/tokens')) {
      return jsonResponse([])
    }
    return jsonResponse([workspace])
  })

  const { screen } = await renderWithRouter(<SettingsPage />)

  await expect.element(screen.getByText(/Events are arriving/)).toBeInTheDocument()
})
