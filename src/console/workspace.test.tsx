import { HttpResponse } from 'msw'
import { beforeEach, expect, test } from 'vitest'

import {
  handleOperatorWorkspaceAuditList,
  handleOperatorWorkspacesGet,
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

// The page always reads the Audit log; tests that care override it.
beforeEach(() => {
  worker.use(handleOperatorWorkspaceAuditList({ body: { items: [] } }))
})

test('lists the audit log with an Operator shown as sphericon staff, and loads more', async () => {
  worker.use(
    handleOperatorWorkspacesGet({ body: HOOLI }),
    handleOperatorWorkspaceAuditList(({ request }) =>
      new URL(request.url).searchParams.get('cursor')
        ? HttpResponse.json({
            items: [
              {
                id: '1',
                occurredAt: '2026-01-01T00:00:00Z',
                actor: { kind: 'user', id: '9', name: 'Peter' },
                action: 'tag.create',
                target: { type: 'tag', id: '12', name: 'vip' },
              },
            ],
          })
        : HttpResponse.json({
            items: [
              {
                id: '2',
                occurredAt: '2026-02-01T00:00:00Z',
                actor: { kind: 'operator', name: 'sphericon staff' },
                action: 'workspace.suspend',
                target: { type: 'workspace', id: '5', name: 'Hooli' },
              },
            ],
            nextCursor: '2',
          }),
    ),
  )
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await expect.element(screen.getByText('sphericon staff')).toBeInTheDocument()
  await expect.element(screen.getByText('workspace.suspend')).toBeInTheDocument()
  await screen.getByRole('button', { name: 'Load more' }).click()
  await expect.element(screen.getByText('tag.create')).toBeInTheDocument()
  await expect.element(screen.getByText('Peter')).toBeInTheDocument()
})

test('an empty audit log says so', async () => {
  worker.use(handleOperatorWorkspacesGet({ body: HOOLI }))
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await expect.element(screen.getByText('No recorded changes')).toBeInTheDocument()
})

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
