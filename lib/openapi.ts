import { schema as openapiSchema } from '../generated/openapi-box.js'

export type OpenapiResponseSchema = {
  'x-status-code'?: string | number
}

type OpenapiOperation = {
  args: { properties: Record<string, unknown> }
  data?: OpenapiResponseSchema
  error?: {
    anyOf?: OpenapiResponseSchema[]
  }
}

type FastifySchema = {
  querystring?: unknown
  body?: unknown
  params?: unknown
  response: Record<number, OpenapiResponseSchema>
}

type OpenapiSchema = typeof openapiSchema

const buildResponseSchema = (
  operation: OpenapiOperation,
): Record<number, OpenapiResponseSchema> => {
  const responseSchema: Record<number, OpenapiResponseSchema> = {}
  const dataSchema = operation.data
  const dataStatus = dataSchema?.['x-status-code']

  if (dataSchema && dataStatus !== undefined) {
    responseSchema[Number(dataStatus)] = dataSchema
  }

  // Error payloads are produced by Fastify/@fastify/sensible and can include
  // framework-specific fields (e.g. message). Keeping runtime validation only
  // for success responses avoids stripping those fields from error bodies.

  return responseSchema
}

export const toFastifySchema = <
  Path extends keyof OpenapiSchema,
  Method extends keyof OpenapiSchema[Path],
>(
  path: Path,
  method: Method,
): FastifySchema => {
  const operation = openapiSchema[path][method] as OpenapiOperation
  const argsProperties = operation.args.properties
  const schema: FastifySchema = {
    response: buildResponseSchema(operation),
  }

  if ('query' in argsProperties) {
    schema.querystring = argsProperties.query
  }

  if ('body' in argsProperties) {
    schema.body = argsProperties.body
  }

  if ('params' in argsProperties) {
    schema.params = argsProperties.params
  }

  return schema
}
