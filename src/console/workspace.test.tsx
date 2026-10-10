import { HttpResponse } from 'msw'
import { expect, test } from 'vitest'

import {
  handleOperatorWorkspacesGet,
  handleOperatorWorkspacesSuspend,
  handleOperatorWorkspacesUnsuspend,
} from '../generated/operator/msw.gen.ts'
import type { OperatorWorkspaceResource } from '../generated/operator/types.gen.ts'
import { consoleWorkspaceRoute } from '../router.tsx'
import { problem } from '../test/problem.ts'
import { renderWithRouter } from '../test/renderWithRouter.tsx'
import { worker } from '../test/worker.ts'
import { ConsoleWorkspacePage } from './workspace.tsx'

// Mounted at the route's id, not its fullPath: the console's guard route is pathless, so the
// two differ and `consoleWorkspaceRoute.useParams()` looks the match up by id.
const MOUNT = {
  path: consoleWorkspaceRoute.id,
  initialPath: consoleWorkspaceRoute.id.replace('$workspaceId', '5'),
}

const HOOLI: OperatorWorkspaceResource = {
  id: '5',
  slug: 'hooli',
  name: 'Hooli',
  createdAt: '2026-01-01T00:00:00Z',
  suspension: {
    at: '2026-02-01T00:00:00Z',
    actor: { kind: 'operator', id: 'op-fixture' },
    reason: 'abuse report',
  },
}

test('shows the metadata and who suspended the workspace, and why', async () => {
  worker.use(handleOperatorWorkspacesGet({ body: HOOLI }))
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await expect.element(screen.getByRole('heading', { name: 'Hooli' })).toBeInTheDocument()
  await expect.element(screen.getByText('hooli', { exact: true })).toBeInTheDocument()
  await expect.element(screen.getByText('Operator op-fixture')).toBeInTheDocument()
  await expect.element(screen.getByText('abuse report')).toBeInTheDocument()
})

test('a workspace that is not suspended says so', async () => {
  worker.use(handleOperatorWorkspacesGet({ body: { ...HOOLI, suspension: null } }))
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await expect.element(screen.getByText('Sending is not suspended.')).toBeInTheDocument()
})

test('a non-operator actor shows its kind, not an id', async () => {
  worker.use(
    handleOperatorWorkspacesGet({
      body: {
        ...HOOLI,
        suspension: { at: '2026-02-01T00:00:00Z', actor: { kind: 'cli' }, reason: null },
      },
    }),
  )
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await expect.element(screen.getByText('Command line')).toBeInTheDocument()
  await expect.element(screen.getByText('No reason given')).toBeInTheDocument()
})

test('an unknown workspace shows the error', async () => {
  worker.use(handleOperatorWorkspacesGet(() => problem(404, { detail: 'workspace not found' })))
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await expect.element(screen.getByText('workspace not found')).toBeInTheDocument()
})

test('suspending needs a reason, and sends it only after the confirm button', async () => {
  const suspended: unknown[] = []
  worker.use(
    handleOperatorWorkspacesGet({ body: { ...HOOLI, suspension: null } }),
    handleOperatorWorkspacesSuspend(async ({ request, params }) => {
      suspended.push({ id: params.workspaceId, body: await request.json() })
      return HttpResponse.json({ changed: true, workspace: HOOLI })
    }),
  )
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await screen.getByRole('button', { name: 'Suspend sending' }).click()
  const dialog = screen.getByRole('dialog')
  await dialog.getByRole('button', { name: 'Suspend sending' }).click()
  await expect.element(dialog.getByText('A reason is required')).toBeInTheDocument()
  expect(suspended).toEqual([])

  await dialog.getByRole('textbox', { name: /Reason/ }).fill('  abuse report ')
  await dialog.getByRole('button', { name: 'Suspend sending' }).click()

  await expect.poll(() => suspended).toEqual([{ id: '5', body: { reason: 'abuse report' } }])
})

test('lifting a suspension asks for confirmation first', async () => {
  const lifted: string[] = []
  worker.use(
    handleOperatorWorkspacesGet({ body: HOOLI }),
    handleOperatorWorkspacesUnsuspend(({ params }) => {
      lifted.push(params.workspaceId)
      return HttpResponse.json({ changed: true, workspace: { ...HOOLI, suspension: null } })
    }),
  )
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await screen.getByRole('button', { name: 'Lift suspension' }).click()
  await expect
    .element(screen.getByRole('dialog').getByText('Lift the suspension?'))
    .toBeInTheDocument()
  expect(lifted).toEqual([])

  await screen.getByRole('dialog').getByRole('button', { name: 'Lift suspension' }).click()

  await expect.poll(() => lifted).toEqual(['5'])
})

test('a no-op says nothing changed', async () => {
  worker.use(
    handleOperatorWorkspacesGet({ body: HOOLI }),
    handleOperatorWorkspacesUnsuspend({ body: { changed: false, workspace: HOOLI } }),
  )
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await screen.getByRole('button', { name: 'Lift suspension' }).click()
  await screen.getByRole('dialog').getByRole('button', { name: 'Lift suspension' }).click()

  await expect
    .element(screen.getByText('Nothing changed: sending was not suspended.'))
    .toBeInTheDocument()
})
