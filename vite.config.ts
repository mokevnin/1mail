import react from '@vitejs/plugin-react'
import { playwright } from '@vitest/browser-playwright'
import type { Plugin } from 'vite'
import { defineConfig } from 'vitest/config'

const SUPPORTED_LOCALES = ['en', 'ru', 'es']

// Dev-only: substitute the {{APP_LOCALE}} sentinels in index.html the same way
// the Go backend does at serve time in prod (internal/server/spa.go). Applied
// only in `serve` so the `build` output keeps the raw sentinels for Go to
// replace at runtime. Drives off APP_LOCALE — the same env var the backend uses.
function devLocalePlugin(): Plugin {
  return {
    name: '1mail:dev-locale',
    apply: 'serve',
    transformIndexHtml(html) {
      const env = process.env.APP_LOCALE ?? 'en'
      const locale = SUPPORTED_LOCALES.includes(env) ? env : 'en'
      return html.replaceAll('{{APP_LOCALE}}', locale)
    },
  }
}

// The app is reached via Caddy at https://1mail.localhost, which terminates TLS
// and is the single place that routes API paths (/site, /collect, /auth,
// /avatar, /api) to the Go backend. Vite serves only the SPA + HMR here.
export default defineConfig({
  plugins: [react(), devLocalePlugin()],
  // Under Vitest browser mode the page loads @vite/client, which would try to open
  // the dev HMR websocket below (wss://1mail.localhost:443 — unreachable in CI) and
  // race the teardown with an unhandled "WebSocket closed without opened" rejection.
  // Disable HMR entirely during tests; keep the Caddy dev config for `make dev`.
  server: process.env.VITEST
    ? { hmr: false }
    : {
        host: true,
        allowedHosts: ['1mail.localhost'],
        // HMR runs through Caddy's HTTPS origin, so the client connects over wss:443.
        hmr: { protocol: 'wss', host: '1mail.localhost', clientPort: 443 },
      },
  test: {
    testTimeout: 10_000,
    setupFiles: ['./src/test/setup.tsx'],
    browser: {
      enabled: true,
      provider: playwright(),
      instances: [{ browser: 'chromium' }],
      headless: true, // the dev/CI container has no display
    },
  },
})
