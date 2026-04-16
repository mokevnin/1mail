import { FormatRegistry, type TSchema } from '@sinclair/typebox'
import { TypeCompiler } from '@sinclair/typebox/compiler'
import { Value } from '@sinclair/typebox/value'
import type { FastifySchemaCompiler, FastifySchemaValidationError } from 'fastify'

function toFastifyValidationErrors(
  errors: ReturnType<ReturnType<typeof TypeCompiler.Compile>['Errors']>,
) {
  return [...errors].map((error): FastifySchemaValidationError => {
    return {
      keyword: `typebox:${error.type}`,
      instancePath: error.path,
      schemaPath: '',
      params: {},
      message: error.message,
    }
  })
}

function registerCommonFormats() {
  if (!FormatRegistry.Has('email')) {
    FormatRegistry.Set('email', (value) => /.+@.+\..+/.test(value))
  }

  if (!FormatRegistry.Has('date-time')) {
    FormatRegistry.Set('date-time', (value) => !Number.isNaN(Date.parse(value)))
  }
}

/**
 * We generate OpenAPI schemas with @sinclair/typebox (openapi-box).
 * The default compiler from @fastify/type-provider-typebox uses `typebox` v1 runtime,
 * so coercion for params/query does not work reliably with our generated schemas.
 */
export const SinclairTypeBoxValidatorCompiler: FastifySchemaCompiler<TSchema> = ({
  schema,
  httpPart,
}) => {
  registerCommonFormats()
  const typeCheck = TypeCompiler.Compile(schema)

  return (value) => {
    const converted = httpPart === 'body' ? value : Value.Convert(schema, value)

    if (typeCheck.Check(converted)) {
      return { value: converted }
    }

    return {
      error: toFastifyValidationErrors(typeCheck.Errors(converted)),
    }
  }
}
