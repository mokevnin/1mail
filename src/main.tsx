import { MantineProvider } from '@mantine/core'
import { ModalsProvider } from '@mantine/modals'
import { Notifications } from '@mantine/notifications'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { StrictMode, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { client } from './generated/site/client.gen.ts'
import './i18n.ts'
import { initTracking } from '@1mail/analytics'
import { router } from './router.tsx'

client.setConfig({ baseUrl: '/site' })

import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
import 'mantine-datatable/styles.css'
import './main.css'

function Root() {
  const [queryClient] = useState(() => new QueryClient())

  return (
    <QueryClientProvider client={queryClient}>
      <MantineProvider>
        <ModalsProvider>
          <Notifications position="top-right" />
          <RouterProvider router={router} />
        </ModalsProvider>
      </MantineProvider>
    </QueryClientProvider>
  )
}

const container = document.getElementById('root')
if (!container) {
  throw new Error('Root element not found')
}

initTracking({
  collectKey: import.meta.env.VITE_COLLECT_SITE_KEY ?? '',
  baseUrl: import.meta.env.VITE_COLLECT_BASE_URL ?? '',
})

createRoot(container).render(
  <StrictMode>
    <Root />
  </StrictMode>,
)
