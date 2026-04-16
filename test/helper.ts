import path from 'node:path'
import type { FastifyInstance } from 'fastify'
import { build as buildApplication } from 'fastify-cli/helper.js'
import type { TestContext } from 'vitest'

const AppPath = path.join(import.meta.dirname, '..', 'app.ts')

// Fill in this config with all the configurations
// needed for testing the application
function config() {
  return {
    skipOverride: true, // Register our application with fastify-plugin
  }
}

// automatically build and tear down our instance
async function build(t: TestContext): Promise<FastifyInstance> {
  // you can set all the options supported by the fastify CLI command
  const argv = [AppPath]

  // fastify-plugin ensures that all decorators
  // are exposed for testing purposes, this is
  // different from the production setup
  const app = (await buildApplication(argv, config())) as FastifyInstance

  // close the app after we are done
  t.onTestFinished(() => app.close())

  return app
}

export { build, config }
