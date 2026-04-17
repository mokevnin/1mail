import fp from 'fastify-plugin'
import { validatorCompiler } from 'fastify-type-provider-zod'

export default fp(async (fastify) => {
  fastify.setValidatorCompiler(validatorCompiler)
})
