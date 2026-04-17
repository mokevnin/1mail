const VISITOR_COOKIE_NAME = 'om_vid'
const FLUSH_DELAY_MS = 200

type IdentifyPayload = {
  email?: string | null
  phone?: string | null
  subjectId?: string | null
  traits?: Record<string, unknown> | null
}

type TrackPayload = {
  visitorId: string
  action: string
  properties?: Record<string, unknown> | null
  occurredAt?: string | null
}

type QueueCommand =
  | ['identify', IdentifyPayload]
  | ['track', string, Record<string, unknown> | null | undefined]

type QueueApi = QueueCommand[] & {
  push: (...items: QueueCommand[]) => number
}

type TrackerConfig = {
  baseUrl: string
  collectKey: string
}

type Runtime = {
  config: TrackerConfig
  visitorId: string
  pendingEvents: TrackPayload[]
  flushTimer: number | null
}

declare global {
  interface Window {
    _omq?: QueueApi
  }
}

let runtime: Runtime | null = null

function getTrackerConfig(): TrackerConfig | null {
  const collectKey = import.meta.env.VITE_COLLECT_SITE_KEY?.trim()
  const baseUrl = (import.meta.env.VITE_COLLECT_BASE_URL?.trim() || '').replace(/\/$/, '')

  if (!collectKey) {
    return null
  }

  return { baseUrl, collectKey }
}

function randomVisitorId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }

  return `vid_${Math.random().toString(36).slice(2)}${Date.now().toString(36)}`
}

function readCookie(name: string): string | null {
  const cookie = document.cookie.split('; ').find((part) => part.startsWith(`${name}=`))

  if (!cookie) {
    return null
  }

  const value = cookie.slice(name.length + 1)

  return value ? decodeURIComponent(value) : null
}

function writeCookie(name: string, value: string): void {
  document.cookie = `${name}=${encodeURIComponent(value)}; Path=/; Max-Age=${60 * 60 * 24 * 365}; SameSite=Lax`
}

function getOrCreateVisitorId(): string {
  const existing = readCookie(VISITOR_COOKIE_NAME)

  if (existing) {
    return existing
  }

  const created = randomVisitorId()
  writeCookie(VISITOR_COOKIE_NAME, created)

  return created
}

function buildHeaders(config: TrackerConfig): HeadersInit {
  return {
    'content-type': 'application/json',
    'x-collect-key': config.collectKey,
  }
}

function collectUrl(config: TrackerConfig, path: string): string {
  return `${config.baseUrl}${path}`
}

function flushSoon(): void {
  if (!runtime || runtime.flushTimer !== null) {
    return
  }

  runtime.flushTimer = window.setTimeout(() => {
    void flushEvents()
  }, FLUSH_DELAY_MS)
}

async function flushEvents(): Promise<void> {
  if (!runtime) {
    return
  }

  if (runtime.flushTimer !== null) {
    window.clearTimeout(runtime.flushTimer)
    runtime.flushTimer = null
  }

  if (runtime.pendingEvents.length === 0) {
    return
  }

  const events = runtime.pendingEvents.splice(0, runtime.pendingEvents.length)

  await fetch(collectUrl(runtime.config, '/collect/events'), {
    method: 'POST',
    headers: buildHeaders(runtime.config),
    body: JSON.stringify({ events }),
    keepalive: true,
  }).catch(() => {
    runtime?.pendingEvents.unshift(...events)
  })
}

function flushEventsOnPageHide(): void {
  if (!runtime || runtime.pendingEvents.length === 0) {
    return
  }

  void flushEvents()
}

function processQueueCommand(command: QueueCommand): void {
  if (command[0] === 'identify') {
    void identify(command[1] ?? {})
    return
  }

  void track(command[1], command[2] ?? null)
}

function createQueueApi(initial: QueueCommand[]): QueueApi {
  const queue = initial as QueueApi

  queue.push = (...items: QueueCommand[]) => {
    for (const item of items) {
      processQueueCommand(item)
    }

    return queue.length
  }

  return queue
}

export function initTracking(): void {
  if (typeof window === 'undefined' || runtime) {
    return
  }

  const config = getTrackerConfig()
  const initialQueue = Array.isArray(window._omq) ? [...window._omq] : []

  window._omq = createQueueApi([])

  if (!config) {
    return
  }

  runtime = {
    config,
    visitorId: getOrCreateVisitorId(),
    pendingEvents: [],
    flushTimer: null,
  }

  for (const command of initialQueue) {
    processQueueCommand(command)
  }

  window.addEventListener('pagehide', flushEventsOnPageHide)
}

export async function identify(payload: IdentifyPayload): Promise<void> {
  if (!runtime) {
    if (typeof window !== 'undefined') {
      if (!window._omq) {
        window._omq = [] as unknown as QueueApi
      }

      const queue = window._omq as unknown as QueueCommand[]
      queue.push(['identify', payload])
    }

    return
  }

  await fetch(collectUrl(runtime.config, '/collect/identify'), {
    method: 'POST',
    headers: buildHeaders(runtime.config),
    body: JSON.stringify({
      visitorId: runtime.visitorId,
      email: payload.email ?? null,
      phone: payload.phone ?? null,
      subjectId: payload.subjectId ?? null,
      traits: payload.traits ?? null,
    }),
    keepalive: true,
  }).catch(() => undefined)
}

export async function track(
  action: string,
  properties?: Record<string, unknown> | null,
): Promise<void> {
  if (!runtime) {
    if (typeof window !== 'undefined') {
      if (!window._omq) {
        window._omq = [] as unknown as QueueApi
      }

      const queue = window._omq as unknown as QueueCommand[]
      queue.push(['track', action, properties])
    }

    return
  }

  runtime.pendingEvents.push({
    visitorId: runtime.visitorId,
    action,
    properties: properties ?? null,
    occurredAt: new Date().toISOString(),
  })

  flushSoon()
}

export function trackPageView(input: {
  path: string
  url: string
  title: string
  referrer: string
  locale: string
}): void {
  void track('page.view', input)
}
