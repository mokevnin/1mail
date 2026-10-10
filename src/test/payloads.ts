// Typed building blocks for the contract-typed `{ body }` responses of the generated MSW handlers.

export const TIMESTAMPS = { createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' }

// page wraps items in the paginated list envelope every list endpoint returns.
export function page<T>(items: T[], totalItems = items.length) {
  return { items, page: 1, pageSize: 25, totalItems, totalPages: Math.ceil(totalItems / 25) }
}
