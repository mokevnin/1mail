import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: './openapi/external.openapi.json',
  output: './generated/handlers',
  plugins: [
    '@hey-api/typescript',
    {
      dates: true,
      name: '@hey-api/transformers',
    },
    'fastify',
    'zod',
  ],
})
