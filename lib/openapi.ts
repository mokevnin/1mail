import openapi from '../openapi/openapi.json' with { type: 'json' }

type OpenApiDocument = typeof openapi
type OpenApiPaths = OpenApiDocument['paths']
type OpenApiPath = keyof OpenApiPaths
type OpenApiMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

type JsonObject = Record<string, unknown>
type ParameterObject = {
  in: 'query' | 'path' | 'header' | 'cookie'
  name: string
  required?: boolean
  schema?: unknown
}

// type RequestBodyObject = {
//   content?: Record<string, { schema?: unknown }>
// }

type FastifySchema = {
  querystring?: unknown
  body?: unknown
  params?: unknown
}

function isObject(value: unknown): value is JsonObject {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function resolveLocalRef(ref: string): unknown {
  if (!ref.startsWith('#/')) {
    throw new Error(`Only local refs are supported: ${ref}`)
  }

  const parts = ref.slice(2).split('/')
  let current: unknown = openapi

  for (const part of parts) {
    if (!isObject(current) || !(part in current)) {
      throw new Error(`Cannot resolve OpenAPI ref: ${ref}`)
    }

    current = current[part]
  }

  return current
}

function resolveSchema(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(resolveSchema)
  }

  if (!isObject(value)) {
    return value
  }

  if (typeof value.$ref === 'string') {
    return resolveSchema(resolveLocalRef(value.$ref))
  }

  const resolved: JsonObject = {}

  for (const [key, nestedValue] of Object.entries(value)) {
    resolved[key] = resolveSchema(nestedValue)
  }

  if (resolved.nullable === true) {
    const { nullable: _nullable, ...nonNullable } = resolved

    return {
      anyOf: [
        {
          type: 'null',
        },
        nonNullable,
      ],
    }
  }

  return resolved
}

function resolveParameter(value: unknown): ParameterObject | undefined {
  const resolved = resolveSchema(value)

  if (!isObject(resolved) || typeof resolved.in !== 'string' || typeof resolved.name !== 'string') {
    return undefined
  }

  return resolved as ParameterObject
}

function buildParameterSchema(parameters: unknown[] | undefined, kind: ParameterObject['in']) {
  const properties: Record<string, unknown> = {}
  const required: string[] = []

  for (const parameterValue of parameters ?? []) {
    const parameter = resolveParameter(parameterValue)

    if (!parameter || parameter.in !== kind || !parameter.schema) {
      continue
    }

    properties[parameter.name] = parameter.schema

    if (parameter.required) {
      required.push(parameter.name)
    }
  }

  if (Object.keys(properties).length === 0) {
    return undefined
  }

  const result: JsonObject = {
    type: 'object',
    properties,
  }

  if (required.length > 0) {
    result.required = required
  }

  return result
}

function resolveRequestBodySchema(operation: JsonObject): unknown {
  const requestBodyValue = operation.requestBody
  const requestBody = resolveSchema(requestBodyValue)

  if (!isObject(requestBody)) {
    return undefined
  }

  const content = requestBody.content
  if (!isObject(content)) {
    return undefined
  }

  const jsonContent = content['application/json']
  if (!isObject(jsonContent)) {
    return undefined
  }

  return jsonContent.schema
}

export const toFastifySchema = <Path extends OpenApiPath>(
  path: Path,
  method: OpenApiMethod,
): FastifySchema => {
  const pathItem = openapi.paths[path] as JsonObject
  const operationValue = pathItem[method.toLowerCase()]

  if (!isObject(operationValue)) {
    throw new Error(`OpenAPI operation not found: ${method} ${String(path)}`)
  }

  const pathParameters = Array.isArray(pathItem.parameters) ? pathItem.parameters : []
  const operationParameters = Array.isArray(operationValue.parameters)
    ? operationValue.parameters
    : []
  const parameters = [...pathParameters, ...operationParameters]

  const schema: FastifySchema = {}
  const querystring = buildParameterSchema(parameters, 'query')
  const params = buildParameterSchema(parameters, 'path')
  const body = resolveRequestBodySchema(operationValue)

  if (querystring) {
    schema.querystring = querystring
  }

  if (params) {
    schema.params = params
  }

  if (body) {
    schema.body = body
  }

  return schema
}
