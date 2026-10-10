// Global setup for Vitest Browser Mode tests. Loaded once per test file via
// `test.setupFiles` in vite.config.ts.
//
// - Mantine styles so rendered components look/behave like the real app.
// - i18n init so `t(($) => $.x)` translation keys resolve to English strings.
import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
import '../i18n.ts'
import { afterAll, afterEach, beforeAll } from 'vitest'

import { client } from '../generated/site/client.gen.ts'
import { worker } from './worker.ts'

// The app pins the generated client's base URL in src/main.tsx; mirror it here.
client.setConfig({ baseUrl: '/site' })

// A request without a handler fails the test loudly; handlers are dropped after every test.
beforeAll(() => worker.start({ onUnhandledRequest: 'error', quiet: true }))
afterEach(() => worker.resetHandlers())
afterAll(() => worker.stop())
