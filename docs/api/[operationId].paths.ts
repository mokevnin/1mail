import { usePaths } from 'vitepress-openapi'

import spec from '../../openapi/external.openapi.json' with { type: 'json' }

export default {
  paths() {
    return usePaths({ spec })
      .getPathsByVerbs()
      .map(({ operationId, summary }: { operationId: string; summary: string }) => ({
        params: { operationId, pageTitle: `${summary} - 1mail API` },
      }))
  },
}
