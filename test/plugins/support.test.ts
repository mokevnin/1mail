import assert from 'node:assert'
import { test } from 'node:test'
import Fastify from 'fastify'
import Support from '../../plugins/support.ts'

test('support works standalone', async (_t) => {
  const fastify = Fastify()
  fastify.register(Support)

  await fastify.ready()
  assert.equal(fastify.someSupport(), 'hugs')
})
