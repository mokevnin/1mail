import { expect, test } from 'vitest'

import type {
  SiteBroadcastResource,
  SiteBroadcastsScheduleData,
  SiteBroadcastsTestSendData,
} from '../../generated/site/types.gen.ts'
import { broadcastsEditRoute, broadcastsReportRoute } from '../../router.tsx'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { routeMount } from '../../test/routeMount.ts'
import { DeliveryBlock } from './DeliveryBlock.tsx'

const ID7 = { slug: 'test', id: '7' }
const MOUNT = routeMount(broadcastsEditRoute, { slug: 'test', broadcastId: '7' })
const SEND_FAILED = 'Failed to send broadcast'

const draft: SiteBroadcastResource = {
  id: '7',
  name: 'Launch',
  subject: 'Hello',
  body: '<mjml></mjml>',
  bodyText: 'Hello',
  status: 'draft',
  stats: {
    recipientsTotal: 0,
    sentCount: 0,
    openedCount: 0,
    clickedCount: 0,
    unsubscribedCount: 0,
    failedCount: 0,
    deliveryRate: 0,
    failureRate: 0,
    openRate: 0,
    clickRate: 0,
    clickToOpenRate: 0,
    unsubscribeRate: 0,
  },
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}

test('scheduling a draft posts the chosen time as an ISO timestamp and opens the report', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    route<SiteBroadcastsScheduleData>(
      'POST',
      '/workspaces/{slug}/broadcasts/{id}/schedule',
      ID7,
      async (req) => {
        bodies.push(await req.json())
        return jsonResponse(draft)
      },
    ),
  ])
  const { screen, navigate } = await renderWithRouter(<DeliveryBlock broadcast={draft} />, MOUNT)

  await screen.getByLabelText('Schedule for').fill('2030-05-06T07:08')
  await screen.getByRole('button', { name: 'Schedule', exact: true }).click()

  await expect.element(screen.getByText('Broadcast scheduled')).toBeInTheDocument()
  expect(bodies).toEqual([{ scheduledAt: new Date('2030-05-06T07:08').toISOString() }])
  await expect
    .poll(() => navigate.mock.calls)
    .toContainEqual([{ to: broadcastsReportRoute.to, params: { slug: 'test', broadcastId: '7' } }])
})

test('a test send posts the typed address and confirms without navigating', async () => {
  const bodies: unknown[] = []
  mockClientRoutes([
    route<SiteBroadcastsTestSendData>(
      'POST',
      '/workspaces/{slug}/broadcasts/{id}/test-send',
      ID7,
      async (req) => {
        bodies.push(await req.json())
        return new Response(null, { status: 204 })
      },
    ),
  ])
  const { screen, navigate } = await renderWithRouter(<DeliveryBlock broadcast={draft} />, MOUNT)

  await screen.getByLabelText('Send a test to').fill('qa@example.com')
  await screen.getByRole('button', { name: 'Send test' }).click()

  await expect.element(screen.getByText('Test email sent')).toBeInTheDocument()
  expect(bodies).toEqual([{ email: 'qa@example.com' }])
  expect(navigate).not.toHaveBeenCalled()
})

test('a rejected test send shows the error toast', async () => {
  mockClientRoutes([
    route<SiteBroadcastsTestSendData>(
      'POST',
      '/workspaces/{slug}/broadcasts/{id}/test-send',
      ID7,
      () => jsonResponse({ title: 'Bad', detail: 'No integration' }, { status: 422 }),
    ),
  ])
  const { screen } = await renderWithRouter(<DeliveryBlock broadcast={draft} />, MOUNT)

  await screen.getByLabelText('Send a test to').fill('qa@example.com')
  await screen.getByRole('button', { name: 'Send test' }).click()

  await expect.element(screen.getByText(SEND_FAILED)).toBeInTheDocument()
})
