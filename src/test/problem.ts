import { HttpResponse } from 'msw'

// problem builds an RFC 7807 problem+json error response, the way the backend renders errors.
// `fields` is the problem title, or the extra members of the problem document (detail,
// retryAfter, ...); the status member is always set.
export function problem(status: number, fields: string | Record<string, unknown> = 'Boom') {
  const members = typeof fields === 'string' ? { title: fields } : fields
  return HttpResponse.json(
    { ...members, status },
    { status, headers: { 'content-type': 'application/problem+json' } },
  )
}
