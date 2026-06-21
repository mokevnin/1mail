// Global setup for Vitest Browser Mode tests. Loaded once per test file via
// `test.setupFiles` in vite.config.ts.
//
// - Mantine styles so rendered components look/behave like the real app.
// - i18n init so `t(($) => $.x)` translation keys resolve to English strings.
import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
import '../i18n.ts'
