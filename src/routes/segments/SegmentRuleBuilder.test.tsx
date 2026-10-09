import { expect, test, vi } from 'vitest'

import type {
  SiteCustomFieldsListData,
  SiteEventsActionsData,
  SiteSegmentsPreviewData,
} from '../../generated/site/types.gen.ts'
import { jsonResponse, mockClientRoutes, route } from '../../test/mockFetch.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { SegmentRuleBuilder } from './SegmentRuleBuilder.tsx'

const SLUG = { slug: 'test' }

function serve(previewBodies: unknown[] = []) {
  mockClientRoutes([
    route<SiteEventsActionsData>('GET', '/workspaces/{slug}/events/actions', SLUG, () =>
      jsonResponse({ actions: ['purchase'] }),
    ),
    route<SiteCustomFieldsListData>('GET', '/workspaces/{slug}/custom-fields', SLUG, () =>
      jsonResponse({
        items: [
          { id: '1', key: 'plan', name: 'Plan', type: 'text' },
          { id: '2', key: 'age', name: 'Age', type: 'number' },
        ],
        totalItems: 2,
      }),
    ),
    route<SiteSegmentsPreviewData>(
      'POST',
      '/workspaces/{slug}/segments/preview',
      SLUG,
      async (req) => {
        previewBodies.push(await req.json())
        return jsonResponse({ count: 3 })
      },
    ),
  ])
}

test('sends the current rule to the audience preview', async () => {
  const previews: unknown[] = []
  serve(previews)
  const value =
    '{"combinator":"and","rules":[{"field":"email","operator":"contains","value":"@x"}]}'
  const { screen } = await renderWithRouter(
    <SegmentRuleBuilder slug="test" value={value} onChange={() => {}} />,
  )

  await screen.getByRole('button', { name: 'Preview audience' }).click()

  await expect.poll(() => previews.length).toBe(1)
  expect(previews).toEqual([{ definition: value }])
  await expect.element(screen.getByText('3 matching contacts')).toBeInTheDocument()
})

test('adding a rule emits the new rule group as JSON', async () => {
  serve()
  const onChange = vi.fn<(json: string) => void>()
  const { screen } = await renderWithRouter(
    <SegmentRuleBuilder slug="test" value="" onChange={onChange} />,
  )

  await screen.getByRole('button', { name: /Rule/ }).first().click()

  await expect.poll(() => onChange.mock.calls.length).toBeGreaterThan(0)
  const [json] = onChange.mock.calls[0] ?? []
  expect(JSON.parse(String(json))).toMatchObject({
    combinator: 'and',
    rules: [{ field: 'subject_id' }],
  })
})

test('an unparsable or non-rule definition falls back to an empty rule group', async () => {
  serve()
  const { screen } = await renderWithRouter(
    <SegmentRuleBuilder slug="test" value="not json" onChange={() => {}} />,
  )

  await expect.element(screen.getByRole('button', { name: 'Preview audience' })).toBeInTheDocument()
})

test('a JSON value that is not a rule group falls back to an empty rule group', async () => {
  serve()
  const { screen } = await renderWithRouter(
    <SegmentRuleBuilder slug="test" value='{"foo":1}' onChange={() => {}} />,
  )

  await expect.element(screen.getByRole('button', { name: 'Preview audience' })).toBeInTheDocument()
})
