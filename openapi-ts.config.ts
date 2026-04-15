import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: './openapi/openapi.yaml',
  output: './types/handlers',
  plugins: ['fastify'],
})
