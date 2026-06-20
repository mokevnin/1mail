import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

// In docker compose the app is reached via Caddy at https://1mail.localhost,
// which terminates TLS and routes API paths to the backend. The proxy below is
// kept as a fallback for direct access to the vite port (http://localhost:5173).
const backend = process.env.BACKEND_URL || 'http://localhost:3300'

export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    allowedHosts: ['1mail.localhost'],
    // HMR runs through Caddy's HTTPS origin, so the client connects over wss:443.
    hmr: { protocol: 'wss', host: '1mail.localhost', clientPort: 443 },
    proxy: {
      '/site': { target: backend, changeOrigin: true },
      '/collect': { target: backend, changeOrigin: true },
      '/auth': { target: backend, changeOrigin: true },
      '/avatar': { target: backend, changeOrigin: true },
    },
  },
  test: {
    testTimeout: 10_000,
  },
})
