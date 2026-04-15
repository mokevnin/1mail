export type OpenapiResponseSchema = {
  'x-status-code'?: string | number
}

export type OpenapiOperation = {
  args: { properties: Record<string, unknown> }
  data?: OpenapiResponseSchema
  error?: {
    anyOf?: OpenapiResponseSchema[]
  }
}

export const buildResponseSchema = (
  operation: OpenapiOperation,
): Record<number, OpenapiResponseSchema> => {
  const responseSchema: Record<number, OpenapiResponseSchema> = {}
  const dataSchema = operation.data
  const dataStatus = dataSchema?.['x-status-code']

  if (dataSchema && dataStatus !== undefined) {
    responseSchema[Number(dataStatus)] = dataSchema
  }

  for (const errorSchema of operation.error?.anyOf ?? []) {
    const errorStatus = errorSchema['x-status-code']

    if (errorStatus !== undefined) {
      responseSchema[Number(errorStatus)] = errorSchema
    }
  }

  return responseSchema
}
