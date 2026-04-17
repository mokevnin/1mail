import type { FastifyInstance, FastifyPluginAsync } from 'fastify'
import { type ApiTokenScope, apiTokenScopes } from '#/db/schema.ts'
import type { RouteHandlers } from '#/generated/handlers/fastify.gen.ts'
import { zAuthTokensDeletePath } from '#/generated/handlers/zod.gen.ts'
import { zCreateApiTokenBodyInput } from '#/lib/http-zod.ts'
import { toApiTokenInfo } from '#/resources/auth.ts'
import {
  createApiToken,
  getApiTokenById,
  listApiTokens,
  requireScope,
  revokeApiToken,
} from '#/use-cases/api-tokens.ts'

function isApiTokenScope(value: string): value is ApiTokenScope {
  return apiTokenScopes.includes(value as ApiTokenScope)
}

function parseScopes(scopes: string[]): ApiTokenScope[] {
  const invalidScope = scopes.find((scope) => !isApiTokenScope(scope))
  if (invalidScope) {
    throw Object.assign(new Error(`Unknown token scope: ${invalidScope}`), { statusCode: 400 })
  }

  return scopes as ApiTokenScope[]
}

function isBootstrapEnabled(): boolean {
  const value = process.env.BOOTSTRAP_ENABLED

  if (value == null) {
    return process.env.NODE_ENV !== 'production'
  }

  return value === 'true'
}

function assertBootstrapHeader(value: string | undefined): void {
  const configured = process.env.BOOTSTRAP_TOKEN

  if (!isBootstrapEnabled() || !configured) {
    throw Object.assign(new Error('Bootstrap is disabled'), { statusCode: 403 })
  }

  if (!value || value !== configured) {
    throw Object.assign(new Error('Invalid bootstrap token'), { statusCode: 401 })
  }
}

const authPlugin: FastifyPluginAsync = async (fastify: FastifyInstance, _opts): Promise<void> => {
  const handlers = {
    authTokensCreate: async (request, reply) => {
      requireScope(request.auth, 'tokens:manage')

      const created = await createApiToken(fastify.db, {
        name: request.body.name,
        scopes: parseScopes(request.body.scopes),
        expiresAt: request.body.expiresAt ?? null,
      })

      return reply.code(201).send({
        token: created.token,
        tokenInfo: toApiTokenInfo(created.record),
      })
    },

    authTokensList: async (request, reply) => {
      requireScope(request.auth, 'tokens:manage')

      const items = await listApiTokens(fastify.db)

      return reply.code(200).send({ items: items.map(toApiTokenInfo) })
    },

    authTokensDelete: async (request, reply) => {
      requireScope(request.auth, 'tokens:manage')

      const id = BigInt(request.params.id)
      const revoked = await revokeApiToken(fastify.db, id)

      fastify.app.guard.found(revoked ? { ...revoked, id: revoked.id } : null)

      return reply.code(204).send()
    },

    authMeGet: async (request, reply) => {
      const token = await getApiTokenById(fastify.db, request.auth?.tokenId ?? 0n)

      fastify.app.guard.found(token ? { id: token.id } : null)
      if (!token) {
        throw new Error('Current token not found')
      }

      return reply.code(200).send(toApiTokenInfo(token))
    },

    authTokensBootstrap: async (request, reply) => {
      assertBootstrapHeader(request.headers['x-bootstrap-token'])

      const created = await createApiToken(fastify.db, {
        name: request.body.name,
        scopes: parseScopes(request.body.scopes),
        expiresAt: request.body.expiresAt ?? null,
      })

      return reply.code(201).send({
        token: created.token,
        tokenInfo: toApiTokenInfo(created.record),
      })
    },
  } satisfies Pick<
    RouteHandlers,
    'authTokensCreate' | 'authTokensList' | 'authTokensDelete' | 'authMeGet' | 'authTokensBootstrap'
  >

  fastify.post(
    '/tokens',
    {
      schema: {
        body: zCreateApiTokenBodyInput,
      },
    },
    handlers.authTokensCreate,
  )
  fastify.get('/tokens', handlers.authTokensList)
  fastify.delete(
    '/tokens/:id',
    {
      schema: {
        params: zAuthTokensDeletePath,
      },
    },
    handlers.authTokensDelete,
  )
  fastify.get('/me', handlers.authMeGet)
  fastify.post(
    '/tokens/bootstrap',
    {
      config: { auth: false },
      schema: {
        body: zCreateApiTokenBodyInput,
      },
    },
    handlers.authTokensBootstrap,
  )
}

export default authPlugin
