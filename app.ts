import path from 'node:path'
import type { AutoloadPluginOptions } from '@fastify/autoload'
import AutoLoad from '@fastify/autoload'
import type { FastifyPluginAsync, FastifyServerOptions } from 'fastify'

export interface AppOptions extends FastifyServerOptions, Partial<AutoloadPluginOptions> {}

const options: AppOptions = {
  // ajv: {
  //   customOptions: {
  //     keywords: ['example'],
  //   },
  // },
}

const app: FastifyPluginAsync<AppOptions> = async (fastify, opts): Promise<void> => {
  // Do not touch the following lines

  // This loads all plugins defined in plugins
  // those should be support plugins that are reused
  // through your application
  fastify.register(AutoLoad, {
    dir: path.join(import.meta.dirname, 'plugins'),
    options: Object.assign({}, opts),
  })

  // This loads public event collection routes used by browser clients
  fastify.register(AutoLoad, {
    dir: path.join(import.meta.dirname, 'server/collect'),
    options: Object.assign({ prefix: '/collect' }, opts),
  })

  // This loads all public HTTP API routes
  fastify.register(AutoLoad, {
    dir: path.join(import.meta.dirname, 'server/http'),
    options: Object.assign({}, opts),
  })
}

export default app
export { app, options }
