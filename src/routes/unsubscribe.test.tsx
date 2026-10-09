import { expect, test } from 'vitest'

import type { SitePublicUnsubscribesPerformData } from '../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../test/mockFetch.ts'
import { renderWithRouter } from '../test/renderWithRouter.tsx'
import { Unsubscribe } from './unsubscribe.tsx'

const perform = (respond: () => Response) =>
  route<SitePublicUnsubscribesPerformData>(
    'POST',
    '/unsubscribes/{token}',
    { token: 'tok-1' },
    respond,
  )

test('pressing Unsubscribe performs the opt-out through the contract and offers unsubscribe-all', async () => {
  const calls: string[] = []
  mockClientRoutes([
    perform(() => {
      calls.push('performed')
      return new Response(null, { status: 204 })
    }),
  ])
  const { screen } = await renderWithRouter(
    <Unsubscribe token="tok-1" all="https://app.example/e/u/everything" />,
  )

  await screen.getByRole('button', { name: 'Unsubscribe' }).click()

  await expect
    .element(screen.getByText('You will no longer receive these emails.'))
    .toBeInTheDocument()
  expect(calls).toEqual(['performed'])
})

test('shows the error state when the token is rejected', async () => {
  mockClientRoutes([
    perform(() => jsonResponse({ status: 400, title: 'Bad Request' }, { status: 400 })),
  ])
  const { screen } = await renderWithRouter(<Unsubscribe token="tok-1" />)

  await screen.getByRole('button', { name: 'Unsubscribe' }).click()

  await expect.element(screen.getByText('Unsubscribe failed')).toBeInTheDocument()
})
