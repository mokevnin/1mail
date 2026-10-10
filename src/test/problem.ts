import { HttpResponse } from 'msw'

// problem builds an RFC 7807 problem+json error response, the way the backend renders errors.
export function problem(status: number, title = 'Boom') {
  return HttpResponse.json(
    { title, status },
    { status, headers: { 'content-type': 'application/problem+json' } },
  )
}
