import { Alert, Button, Card, Stack, Text, Title } from '@mantine/core'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { confirmRoute } from '../router.tsx'

// Public double opt-in confirmation page (ADR 0013). The GET /e/confirm/{token}
// endpoint redirects here and records nothing — the confirmation happens only when
// the user presses Confirm, which POSTs back to the same token endpoint. This keeps
// GET safe so email link scanners can't confirm anyone (a deliberate human act is
// required for legal validity). An expired/invalid token arrives here without a
// `token` (with `expired=1`), so the page offers "sign up again" instead of a dead
// button. The `token` is a public, signed tracking token, so a direct fetch to
// /e/confirm/{token} is used (tracking endpoints are outside the generated client).
export function ConfirmSubscriptionPage() {
  const { t } = useTranslation()
  const { token, expired } = confirmRoute.useSearch()
  const [status, setStatus] = useState<'idle' | 'pending' | 'done' | 'error'>('idle')

  const confirm = async () => {
    setStatus('pending')
    try {
      const res = await fetch(`/e/confirm/${token}`, { method: 'POST' })
      setStatus(res.ok ? 'done' : 'error')
    } catch {
      setStatus('error')
    }
  }

  const isExpired = expired === '1' || !token

  return (
    <Stack maw={460} mx="auto" mt="xl" align="center">
      <Card withBorder w="100%" padding="xl">
        {isExpired ? (
          <Stack align="center" gap="sm">
            <Title order={3}>{t(($) => $.confirmSubscription.expiredTitle)}</Title>
            <Text c="dimmed" ta="center">
              {t(($) => $.confirmSubscription.expiredBody)}
            </Text>
          </Stack>
        ) : status === 'done' ? (
          <Stack align="center" gap="sm">
            <Title order={3}>{t(($) => $.confirmSubscription.doneTitle)}</Title>
            <Text c="dimmed" ta="center">
              {t(($) => $.confirmSubscription.doneBody)}
            </Text>
          </Stack>
        ) : (
          <Stack align="center" gap="md">
            <Title order={3}>{t(($) => $.confirmSubscription.title)}</Title>
            <Text c="dimmed" ta="center">
              {t(($) => $.confirmSubscription.body)}
            </Text>
            {status === 'error' ? (
              <Alert color="red" title={t(($) => $.confirmSubscription.errorTitle)} w="100%">
                {t(($) => $.confirmSubscription.errorBody)}
              </Alert>
            ) : null}
            <Button onClick={confirm} loading={status === 'pending'}>
              {t(($) => $.confirmSubscription.confirm)}
            </Button>
          </Stack>
        )}
      </Card>
    </Stack>
  )
}
