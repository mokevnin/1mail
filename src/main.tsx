import { MantineProvider } from '@mantine/core'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { httpBatchLink } from '@trpc/client'
import { StrictMode, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { trpc } from '../trpc/client.ts'
import App from './App.tsx'

import '@mantine/core/styles.css'
import './main.css'

function Root() {
  const [queryClient] = useState(() => new QueryClient())
  const [trpcClient] = useState(() =>
    trpc.createClient({
      links: [
        httpBatchLink({
          url: 'http://localhost:3000/trpc',
        }),
      ],
    }),
  )

  return (
    <trpc.Provider client={trpcClient} queryClient={queryClient}>
      <QueryClientProvider client={queryClient}>
        <MantineProvider>
          <App />
        </MantineProvider>
      </QueryClientProvider>
    </trpc.Provider>
  )
}

const container = document.getElementById('root')
if (!container) {
  throw new Error('Root element not found')
}

createRoot(container).render(
  <StrictMode>
    <Root />
  </StrictMode>,
)
