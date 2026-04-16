import fastifyAuth from '@fastify/auth'
import fp from 'fastify-plugin'
import { verifyApiToken } from '../use-cases/api-tokens.ts'

type AuthError = Error & {
  statusCode: number
}

function createAuthError(statusCode: number, message: string): AuthError {
  return Object.assign(new Error(message), { statusCode })
}

function extractBearerToken(value: string | undefined): string | undefined {
  if (!value) {
    return undefined
  }

  const [scheme, token, extra] = value.split(' ')
  if (scheme !== 'Bearer' || !token || extra) {
    return undefined
  }

  return token
}

export default fp(
  async (fastify) => {
    await fastify.register(fastifyAuth)

    fastify.decorateRequest('auth', null)

    fastify.decorate('verifyApiToken', async function verifyApiTokenDecorator(request) {
      const tokenValue = extractBearerToken(request.headers.authorization)

      if (!tokenValue) {
        throw createAuthError(401, 'Missing or invalid bearer token')
      }

      const token = await verifyApiToken(fastify.db, tokenValue)
      if (!token) {
        throw createAuthError(401, 'Invalid API token')
      }

      request.auth = {
        tokenId: token.id,
        name: token.name,
        scopes: token.scopes,
      }
    })

    fastify.decorate('requireAuth', fastify.auth([fastify.verifyApiToken]))

    fastify.addHook('onRoute', (routeOptions) => {
      if (routeOptions.url.startsWith('/trpc')) {
        return
      }

      if (routeOptions.config?.auth === false) {
        return
      }

      const currentPreHandler = routeOptions.preHandler
      const authPreHandler = fastify.requireAuth

      routeOptions.preHandler = currentPreHandler
        ? [authPreHandler].concat(currentPreHandler)
        : [authPreHandler]
    })
  },
  {
    name: 'auth',
    dependencies: ['drizzle'],
  },
)
