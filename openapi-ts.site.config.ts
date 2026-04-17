import { $, defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: './openapi/site.openapi.json',
  output: {
    importFileExtension: '.ts',
    path: './generated/site',
  },
  plugins: [
    '@hey-api/typescript',
    {
      dates: true,
      name: '@hey-api/transformers',
    },
    {
      name: 'zod',
      '~resolvers': {
        string: ({ schema, symbols }) => {
          if (schema.format === 'date-time') {
            return $(symbols.z).attr('date').call()
          }
        },
      },
    },
    {
      name: 'orpc',
      validator: 'zod',
    },
  ],
})
