import { createORPCClient } from '@orpc/client'
import { RPCLink } from '@orpc/client/fetch'
import type { RouterClient } from '@orpc/server'
import { createTanstackQueryUtils } from '@orpc/tanstack-query'

import type { OrpcRouter } from '../../server/orpc/router.ts'

const link = new RPCLink({
  url: 'http://localhost:3000/orpc',
})

const client: RouterClient<OrpcRouter> = createORPCClient(link)

export const orpc = createTanstackQueryUtils(client)
