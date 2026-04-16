import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: './openapi/openapi.json',
  output: './generated/handlers',
  plugins: ['fastify'],
})
