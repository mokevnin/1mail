import { defineConfig } from 'oxfmt'

export default defineConfig({
  printWidth: 100,
  singleQuote: true,
  jsxSingleQuote: false,
  semi: false,
  trailingComma: 'all',
  sortImports: {},
  sortPackageJson: true,
  ignorePatterns: [
    'dist',
    'src/generated',
    'types/resources.d.ts',
    'packages/*/src/generated',
    'packages/*/dist',
    'internal/server/assets',
    'gen',
    'ent',
    'pnpm-lock.yaml',
    'skills-lock.json',
    '.agents',
    '.claude',
    '.cache',
    '.vitest',
    'tmp',
  ],
})
