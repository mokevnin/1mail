import { afterEach } from 'vitest'

import { client } from '../generated/site/client.gen.ts'

// The generated hey-api client is a singleton whose Config accepts a `fetch`
// override (see generated/site/client/types.gen.ts). Tests stub the network at
// that seam — no MSW / service worker needed. Always restore the real config
// after each test so stubs don't leak across files.
const DEFAULT_BASE_URL = '/site'

type JsonResponseInit = {
  status?: number
  headers?: Record<string, string>
}

// jsonResponse builds a fetch Response with a JSON body, the way the backend
// returns it (success payloads and RFC 7807 problem+json errors alike).
export function jsonResponse(body: unknown, init: JsonResponseInit = {}): Response {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: { 'content-type': 'application/json', ...init.headers },
  })
}

// mockClientFetch points the generated client at a stub for the duration of a
// test. `handler` receives the same args as global fetch and returns a Response.
export function mockClientFetch(
  handler: (input: RequestInfo | URL, init?: RequestInit) => Response | Promise<Response>,
) {
  const fetchStub: typeof fetch = (input, init) => Promise.resolve(handler(input, init))
  client.setConfig({ baseUrl: DEFAULT_BASE_URL, fetch: fetchStub })
}

afterEach(() => {
  client.setConfig({ baseUrl: DEFAULT_BASE_URL, fetch: globalThis.fetch })
})

type Handler = (req: Request) => Response | Promise<Response>

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

type OperationData = { url: string; path: Record<string, string> }

export type Route = { method: HttpMethod; pathname: () => string; respond: Handler }

// requestOf normalises the args of a fetch call into a Request.
export function requestOf(input: RequestInfo | URL, init?: RequestInit) {
  return input instanceof Request ? input : new Request(String(input), init)
}

// route mocks one API operation. `D` is the generated operation's Data type
// (e.g. SiteBroadcastsListData): its `url` literal type makes the template a
// compile-time checked copy of the contract, and the URL is built by the
// generated client itself from the template and path params, so a renamed or
// moved endpoint fails type-checking instead of silently never matching.
export function route<D extends OperationData>(
  method: HttpMethod,
  url: D['url'],
  path: D['path'],
  respond: Handler,
): Route {
  // Built lazily: the client's baseUrl is only pinned once the stub is installed.
  const pathname = () =>
    new URL(
      client.buildUrl<OperationData>({ baseUrl: client.getConfig().baseUrl, url, path }),
      location.origin,
    ).pathname
  return { method, pathname, respond }
}

// mockClientRoutes serves the given operations and fails loudly on any other
// request, so a test never passes by accident against an unmocked endpoint.
export function mockClientRoutes(routes: Route[]) {
  mockClientFetch((input, init) => {
    const req = requestOf(input, init)
    const { pathname } = new URL(req.url)
    const hit = routes.find((r) => r.method === req.method && r.pathname() === pathname)
    if (!hit) throw new Error(`unmocked request: ${req.method} ${pathname}`)
    return hit.respond(req)
  })
}
