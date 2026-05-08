import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/site': { target: 'http://localhost:3300', changeOrigin: true },
      '/collect': { target: 'http://localhost:3300', changeOrigin: true },
      '/auth': { target: 'http://localhost:3300', changeOrigin: true },
      '/avatar': { target: 'http://localhost:3300', changeOrigin: true },
    },
  },
  test: {
    testTimeout: 10_000,
  },
})
