import path from 'node:path'
import type { AutoloadPluginOptions } from '@fastify/autoload'
import AutoLoad from '@fastify/autoload'
import type { FastifyPluginAsync, FastifyServerOptions } from 'fastify'
import { SinclairTypeBoxValidatorCompiler } from './lib/typebox-validator.ts'

export interface AppOptions extends FastifyServerOptions, Partial<AutoloadPluginOptions> {}

const options: AppOptions = {
  // ajv: {
  //   customOptions: {
  //     keywords: ['example'],
  //   },
  // },
}

const app: FastifyPluginAsync<AppOptions> = async (fastify, opts): Promise<void> => {
  // Place here your custom code!
  fastify.setValidatorCompiler(SinclairTypeBoxValidatorCompiler)

  // Do not touch the following lines

  // This loads all plugins defined in plugins
  // those should be support plugins that are reused
  // through your application
  fastify.register(AutoLoad, {
    dir: path.join(import.meta.dirname, 'plugins'),
    options: Object.assign({}, opts),
  })

  // This loads all plugins defined in routes
  // define your routes in one of these
  fastify.register(AutoLoad, {
    dir: path.join(import.meta.dirname, 'routes'),
    options: Object.assign({}, opts),
  })
}

export default app
export { app, options }
