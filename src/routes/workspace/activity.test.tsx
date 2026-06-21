import { expect, test } from 'vitest'
import { jsonResponse, mockClientFetch } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { ActivityPage } from './activity.tsx'

test('renders the event feed', async () => {
  mockClientFetch(() =>
    jsonResponse({
      items: [
        {
          id: '2',
          subjectId: 'user:bob@example.com',
          email: 'bob@example.com',
          action: 'purchase',
          createdAt: '2026-01-02T00:00:00Z',
        },
      ],
      page: 1,
      pageSize: 25,
      totalItems: 1,
      totalPages: 1,
    }),
  )

  const { screen } = await renderWithRouter(<ActivityPage />)

  await expect.element(screen.getByText('purchase')).toBeInTheDocument()
  await expect.element(screen.getByText('bob@example.com')).toBeInTheDocument()
})
