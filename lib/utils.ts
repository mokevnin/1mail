export const toTimestamp = (value: Date | string): string =>
  value instanceof Date ? value.toISOString() : new Date(value).toISOString()
