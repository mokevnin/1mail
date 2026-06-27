import react from '@vitejs/plugin-react'
import { playwright } from '@vitest/browser-playwright'
import { defineConfig } from 'vitest/config'

// The app is reached via Caddy at https://1mail.localhost, which terminates TLS
// and is the single place that routes API paths (/site, /collect, /auth,
// /avatar, /api) to the Go backend. Vite serves only the SPA + HMR here.
export default defineConfig({
  plugins: [react()],
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
