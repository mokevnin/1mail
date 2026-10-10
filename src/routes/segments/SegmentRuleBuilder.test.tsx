import { HttpResponse } from 'msw'
import { expect, test, vi } from 'vitest'

import {
  handleSiteCustomFieldsList,
  handleSiteEventsActions,
  handleSiteSegmentsPreview,
} from '../../generated/site/msw.gen.ts'
import { page, TIMESTAMPS } from '../../test/payloads.ts'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { worker } from '../../test/worker.ts'
import { SegmentRuleBuilder } from './SegmentRuleBuilder.tsx'

function serve(previewBodies: unknown[] = []) {
  worker.use(
    handleSiteEventsActions({ body: { actions: ['purchase'] } }),
    handleSiteCustomFieldsList({
      body: page([
        { id: '1', key: 'plan', name: 'Plan', type: 'string', ...TIMESTAMPS },
        { id: '2', key: 'age', name: 'Age', type: 'number', ...TIMESTAMPS },
      ]),
    }),
    handleSiteSegmentsPreview(async ({ request }) => {
      previewBodies.push(await request.json())
      return HttpResponse.json({ count: 3 })
    }),
  )
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
