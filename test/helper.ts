import path from 'node:path'
import type { FastifyInstance } from 'fastify'
import { build as buildApplication } from 'fastify-cli/helper.js'
import type { TestContext } from 'vitest'

const AppPath = path.join(import.meta.dirname, '..', 'app.ts')
const DefaultBootstrapToken = 'test-bootstrap-token'

type BuildOptions = {
  bootstrapEnabled?: boolean
  bootstrapToken?: string
}

// Fill in this config with all the configurations
// needed for testing the application
function config() {
  return {
    skipOverride: true, // Register our application with fastify-plugin
  }
}

// automatically build and tear down our instance
async function build(t: TestContext, options: BuildOptions = {}): Promise<FastifyInstance> {
  const previousBootstrapEnabled = process.env.BOOTSTRAP_ENABLED
  const previousBootstrapToken = process.env.BOOTSTRAP_TOKEN

  process.env.BOOTSTRAP_ENABLED = String(options.bootstrapEnabled ?? true)
  process.env.BOOTSTRAP_TOKEN = options.bootstrapToken ?? DefaultBootstrapToken

  // you can set all the options supported by the fastify CLI command
  const argv = [AppPath]

  // fastify-plugin ensures that all decorators
  // are exposed for testing purposes, this is
  // different from the production setup
  const app = (await buildApplication(argv, config())) as FastifyInstance

  // close the app after we are done
  t.onTestFinished(async () => {
    process.env.BOOTSTRAP_ENABLED = previousBootstrapEnabled
    process.env.BOOTSTRAP_TOKEN = previousBootstrapToken

    await app.close()
  })

  return app
}

function authHeader(token: string) {
  return {
    authorization: `Bearer ${token}`,
  }
}

async function bootstrapApiToken(
  app: FastifyInstance,
  input: {
    name?: string
    scopes?: string[]
    expiresAt?: string | null
    bootstrapToken?: string
  } = {},
): Promise<Awaited<ReturnType<FastifyInstance['inject']>>> {
  const response = await app.inject({
    method: 'POST',
    url: '/auth/tokens/bootstrap',
    headers: {
      'x-bootstrap-token':
        input.bootstrapToken ?? process.env.BOOTSTRAP_TOKEN ?? DefaultBootstrapToken,
    },
    payload: {
      name: input.name ?? 'Test token',
      scopes: input.scopes ?? ['contacts:read', 'contacts:write', 'tokens:manage'],
      expiresAt: input.expiresAt ?? null,
    },
  })

  return response
}

export { authHeader, bootstrapApiToken, build, config, DefaultBootstrapToken }
