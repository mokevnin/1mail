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
