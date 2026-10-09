import type { KnipConfig } from 'knip'

const config: KnipConfig = {
  ignore: ['src/generated/**', 'packages/analytics/src/generated/**', 'internal/server/assets/**'],
  ignoreDependencies: [
    '@1mail/analytics',
    '@jsonforms/core',
    '@jsonforms/react',
    '@typespec/.*',
    'npm-check-updates',
  ],
  ignoreExportsUsedInFile: true,
  workspaces: {
    '.': {
      entry: ['i18next.config.ts', 'docs/.vitepress/config.ts'],
    },
  },
}

export default config
