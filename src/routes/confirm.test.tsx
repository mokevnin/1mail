import { expect, test } from 'vitest'

import type { SitePublicConfirmationsPerformData } from '../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../test/mockFetch.ts'
import { renderWithRouter } from '../test/renderWithRouter.tsx'
import { ConfirmSubscription } from './confirm.tsx'

const perform = (respond: () => Response) =>
  route<SitePublicConfirmationsPerformData>(
    'POST',
    '/confirmations/{token}',
    { token: 'tok-1' },
    respond,
  )

test('pressing Confirm performs the confirmation through the contract and thanks the user', async () => {
  const calls: string[] = []
  mockClientRoutes([
    perform(() => {
      calls.push('performed')
      return new Response(null, { status: 204 })
    }),
  ])
  const { screen } = await renderWithRouter(<ConfirmSubscription token="tok-1" />)

  await screen.getByRole('button', { name: 'Confirm subscription' }).click()

  await expect.element(screen.getByText('Subscription confirmed')).toBeInTheDocument()
  expect(calls).toEqual(['performed'])
})

test('shows the error state when the token is rejected', async () => {
  mockClientRoutes([
    perform(() => jsonResponse({ status: 400, title: 'Bad Request' }, { status: 400 })),
  ])
  const { screen } = await renderWithRouter(<ConfirmSubscription token="tok-1" />)

  await screen.getByRole('button', { name: 'Confirm subscription' }).click()

  await expect.element(screen.getByText('Confirmation failed')).toBeInTheDocument()
})

test('an expired link offers sign-up-again instead of a button', async () => {
  const { screen } = await renderWithRouter(<ConfirmSubscription expired />)

  await expect.element(screen.getByText('Link expired')).toBeInTheDocument()
  await expect
    .element(screen.getByRole('button', { name: 'Confirm subscription' }))
    .not.toBeInTheDocument()
})
