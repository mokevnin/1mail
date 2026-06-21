import react from '@vitejs/plugin-react'
import { playwright } from '@vitest/browser-playwright'
import { defineConfig } from 'vitest/config'

// The app is reached via Caddy at https://1mail.localhost, which terminates TLS
// and is the single place that routes API paths (/site, /collect, /auth,
// /avatar, /api) to the Go backend. Vite serves only the SPA + HMR here.
export default defineConfig({
  plugins: [react()],
  server: {
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
