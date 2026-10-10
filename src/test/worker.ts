import { setupWorker } from 'msw/browser'

// The MSW service-worker worker of this test file (started and reset in setup.tsx). A test
// installs the generated `handle*` handlers it needs with `worker.use(...)`.
export const worker = setupWorker()
