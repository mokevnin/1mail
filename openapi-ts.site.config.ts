import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: './openapi/site.openapi.json',
  output: {
    importFileExtension: '.ts',
    path: './src/generated/site',
  },
  plugins: [
    '@hey-api/typescript',
    '@hey-api/client-fetch',
    {
      name: '@tanstack/react-query',
      queryOptions: true,
      mutationOptions: true,
    },
  ],
})
