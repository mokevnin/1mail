import { expect, test } from 'vitest'
import { build } from '../helper.ts'

test('default root route', async (t) => {
  const app = await build(t)

  const res = await app.inject({
    url: '/',
  })
  expect(JSON.parse(res.payload)).toEqual({ root: true })
})
