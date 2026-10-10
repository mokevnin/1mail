import { afterEach, expect, test, vi } from 'vitest'

import { activityRoute } from '../../router.tsx'
import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { SettingsPage } from './settings.tsx'

const workspace = {
  id: '1',
  name: 'Acme',
  slug: 'test',
  collectKey: 'omck_test_key',
  ingestKey: 'omik_test_key',
  postalAddress: '',
  createdAt: '2026-01-01T00:00:00Z',
}

afterEach(() => {
  vi.restoreAllMocks()
})

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
    // The settings page also lists API tokens and integrations (none here).
    if (req.url.includes('/tokens') || req.url.includes('/integrations')) {
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

  const nameInput = screen.getByLabelText(/^Workspace name/)
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
    if (req.url.includes('/tokens') || req.url.includes('/integrations')) {
      return jsonResponse([])
    }
    return jsonResponse([workspace])
  })

  const { screen } = await renderWithRouter(<SettingsPage />)

  await expect.element(screen.getByText(/Events are arriving/)).toBeInTheDocument()
})

function serveWorkspace() {
  mockClientFetch((input) => {
    const req = input instanceof Request ? input : new Request(String(input))
    if (req.url.includes('/events')) {
      return eventsPage(0)
    }
    if (req.url.includes('/tokens') || req.url.includes('/integrations')) {
      return jsonResponse([])
    }
    return jsonResponse([workspace])
  })
}

test('the activity link opens the workspace activity feed', async () => {
  serveWorkspace()
  const { screen, navigate } = await renderWithRouter(<SettingsPage />)

  await screen.getByRole('button', { name: 'Open activity feed' }).click()

  expect(navigate).toHaveBeenCalledWith({ to: activityRoute.to, params: { slug: 'test' } })
})

test('each copy button flips to the copied state', async () => {
  vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue()
  serveWorkspace()
  const { screen } = await renderWithRouter(<SettingsPage />)
  await expect.element(screen.getByText(/omck_test_key/).first()).toBeInTheDocument()

  const copyButtons = screen.getByRole('button', { name: 'Copy', exact: true })
  const total = copyButtons.elements().length
  expect(total).toBeGreaterThanOrEqual(3)
  for (let i = 0; i < total; i++) {
    await copyButtons.first().click()
  }

  await expect
    .poll(() => screen.getByRole('button', { name: 'Copied' }).elements().length)
    .toBe(total)
})

test('shows an error alert when the workspaces fail to load', async () => {
  mockClientFetch(() => jsonResponse({ status: 500, detail: 'boom' }, { status: 500 }))
  const { screen } = await renderWithRouter(<SettingsPage />)

  await expect.element(screen.getByRole('alert', { name: /Failed to load/ })).toBeInTheDocument()
})
