import type { FastifyPluginOptions } from 'fastify'
import fp from 'fastify-plugin'

export type SupportPluginOptions = FastifyPluginOptions

// The use of fastify-plugin is required to be able
// to export the decorators to the outer scope

export default fp<SupportPluginOptions>(async (fastify, _opts) => {
  fastify.decorate('someSupport', () => 'hugs')
})
