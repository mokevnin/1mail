import { expect, test } from 'vitest'

import { handleOperatorWorkspacesGet } from '../generated/operator/msw.gen.ts'
import type { OperatorWorkspaceDetailResource } from '../generated/operator/types.gen.ts'
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

const HOOLI: OperatorWorkspaceDetailResource = {
  id: '5',
  slug: 'hooli',
  name: 'Hooli',
  createdAt: '2026-01-01T00:00:00Z',
  suspension: {
    at: '2026-02-01T00:00:00Z',
    actor: { kind: 'operator', id: 'op-fixture' },
    reason: 'abuse report',
  },
  deliverability: { windowHours: 24, volumeFloor: 1000, sendVolume: 0, domains: [] },
}

const ABOVE_FLOOR = {
  sendingDomainId: '400',
  domain: 'mail.soylent.test',
  complaintRate: { numerator: 3, denominator: 1200, rate: 0.0025 },
  bounceRate: { numerator: 60, denominator: 1300, rate: 0.046 },
}

const BELOW_FLOOR = {
  sendingDomainId: '401',
  domain: 'promo.soylent.test',
  complaintRate: { numerator: 1, denominator: 7, rate: null },
  bounceRate: { numerator: 1, denominator: 8, rate: null },
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

test('shows the send volume and the rates with their counts per sending domain', async () => {
  worker.use(
    handleOperatorWorkspacesGet({
      body: {
        ...HOOLI,
        deliverability: {
          windowHours: 24,
          volumeFloor: 1000,
          sendVolume: 1308,
          domains: [ABOVE_FLOOR],
        },
      },
    }),
  )
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await expect.element(screen.getByText('mail.soylent.test')).toBeInTheDocument()
  await expect.element(screen.getByText('0.25%')).toBeInTheDocument()
  await expect.element(screen.getByText('3 / 1200')).toBeInTheDocument()
  await expect.element(screen.getByText('4.6%')).toBeInTheDocument()
  await expect.element(screen.getByText('1,308')).toBeInTheDocument()
  await expect.element(screen.getByText('Last 24 hours')).toBeInTheDocument()
})

test('a rate below the volume floor shows the notice and its counts, never a percentage', async () => {
  worker.use(
    handleOperatorWorkspacesGet({
      body: {
        ...HOOLI,
        deliverability: {
          windowHours: 24,
          volumeFloor: 1000,
          sendVolume: 8,
          domains: [BELOW_FLOOR],
        },
      },
    }),
  )
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await expect.element(screen.getByText('promo.soylent.test')).toBeInTheDocument()
  await expect
    .element(screen.getByText('Not enough data (fewer than 1000 sent)').first())
    .toBeInTheDocument()
  await expect.element(screen.getByText('1 / 8')).toBeInTheDocument()
  await expect.element(screen.getByText('1 / 7')).toBeInTheDocument()
})

test('a workspace with no sending domains says so', async () => {
  worker.use(handleOperatorWorkspacesGet({ body: HOOLI }))
  const { screen } = await renderWithRouter(<ConsoleWorkspacePage />, MOUNT)

  await expect.element(screen.getByText('No sending domains.')).toBeInTheDocument()
})
