import type { FastifyInstance, FastifyPluginAsync } from 'fastify'
import { type ApiTokenScope, apiTokenScopes } from '../../../db/schema.ts'
import type { RouteHandlers } from '../../../generated/handlers/fastify.gen.ts'
import { toFastifySchema } from '../../../lib/openapi.ts'
import {
  type ApiTokenMetadata,
  createApiToken,
  getApiTokenById,
  listApiTokens,
  requireScope,
  revokeApiToken,
} from '../../../use-cases/api-tokens.ts'

function toApiTokenResponse(token: ApiTokenMetadata) {
  return {
    id: token.id.toString(),
    name: token.name,
    scopes: token.scopes,
    expiresAt: token.expiresAt?.toISOString() ?? null,
    revokedAt: token.revokedAt?.toISOString() ?? null,
    lastUsedAt: token.lastUsedAt?.toISOString() ?? null,
    createdAt: token.createdAt.toISOString(),
    updatedAt: token.updatedAt.toISOString(),
  }
}

function parseOptionalDate(value: string | null | undefined): Date | null {
  if (!value) {
    return null
  }

  return new Date(value)
}

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
        expiresAt: parseOptionalDate(request.body.expiresAt),
      })

      return reply.code(201).send({
        token: created.token,
        tokenInfo: toApiTokenResponse(created.record),
      })
    },

    authTokensList: async (request, reply) => {
      requireScope(request.auth, 'tokens:manage')

      const items = await listApiTokens(fastify.db)

      return reply.code(200).send({ items: items.map(toApiTokenResponse) })
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

      return reply.code(200).send(toApiTokenResponse(token))
    },

    authTokensBootstrap: async (request, reply) => {
      assertBootstrapHeader(request.headers['x-bootstrap-token'])

      const created = await createApiToken(fastify.db, {
        name: request.body.name,
        scopes: parseScopes(request.body.scopes),
        expiresAt: parseOptionalDate(request.body.expiresAt),
      })

      return reply.code(201).send({
        token: created.token,
        tokenInfo: toApiTokenResponse(created.record),
      })
    },
  } satisfies Pick<
    RouteHandlers,
    'authTokensCreate' | 'authTokensList' | 'authTokensDelete' | 'authMeGet' | 'authTokensBootstrap'
  >

  fastify.post(
    '/tokens',
    {
      schema: toFastifySchema('/auth/tokens', 'POST'),
    },
    handlers.authTokensCreate,
  )
  fastify.get(
    '/tokens',
    {
      schema: toFastifySchema('/auth/tokens', 'GET'),
    },
    handlers.authTokensList,
  )
  fastify.delete(
    '/tokens/:id',
    {
      schema: toFastifySchema('/auth/tokens/{id}', 'DELETE'),
    },
    handlers.authTokensDelete,
  )
  fastify.get(
    '/me',
    {
      schema: toFastifySchema('/auth/me', 'GET'),
    },
    handlers.authMeGet,
  )
  fastify.post(
    '/tokens/bootstrap',
    {
      config: { auth: false },
      schema: toFastifySchema('/auth/tokens/bootstrap', 'POST'),
    },
    handlers.authTokensBootstrap,
  )
}

export default authPlugin
