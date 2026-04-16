import { defineConfig } from 'i18next-cli'

export default defineConfig({
  locales: ['en'],
  extract: {
    input: ['src/**/*.{ts,tsx}'],
    output: 'locales/{{language}}/{{namespace}}.json',
    defaultNS: 'translation',
    keySeparator: '.',
  },
  types: {
    input: 'locales/en/**/*.json',
    basePath: 'locales/en',
    output: 'types/i18next.d.ts',
    enableSelector: 'optimize',
  },
})
