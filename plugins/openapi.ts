import path from 'node:path'
import openapiGlue from 'fastify-openapi-glue'
import fp from 'fastify-plugin'
import createHandlers from '../src/api/handlers.ts'

export default fp(async (fastify) => {
  const handlers = createHandlers(fastify)

  const options = {
    specification: path.join(import.meta.dirname, '../openapi/openapi.yaml'),
    serviceHandlers: handlers,
  }

  // Register with host constraint for "api" subdomain
  // In development, you might want to use api.localhost
  // In production, it would be api.yourdomain.com
  fastify.register(
    async (instance) => {
      instance.register(openapiGlue, options)
    },
    {
      constraints: {
        host: /api\..*/, // Matches any subdomain starting with "api."
      },
    },
  )
})
