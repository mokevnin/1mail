import { Alert, Anchor, Button, Card, Stack, Text, Title } from '@mantine/core'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { unsubscribeRoute } from '../router.tsx'

// Public unsubscribe confirmation page (ADR 0012 / RFC 8058). The GET
// /e/u/{token} endpoint redirects here and records nothing — the opt-out happens
// only when the user presses Confirm, which POSTs back to the same token endpoint.
// This keeps GET safe so email link scanners can't unsubscribe anyone. The `token`
// is a public, signed tracking token, so a direct fetch to /e/u/{token} is used
// (tracking endpoints are intentionally outside the generated site client).
export function UnsubscribePage() {
  const { t } = useTranslation()
  const { token, all } = unsubscribeRoute.useSearch()
  const [status, setStatus] = useState<'idle' | 'pending' | 'done' | 'error'>('idle')

  const confirm = async () => {
    setStatus('pending')
    try {
      const res = await fetch(`/e/u/${token}`, { method: 'POST' })
      setStatus(res.ok ? 'done' : 'error')
    } catch {
      setStatus('error')
    }
  }

  return (
    <Stack maw={460} mx="auto" mt="xl" align="center">
      <Card withBorder w="100%" padding="xl">
        {status === 'done' ? (
          <Stack align="center" gap="sm">
            <Title order={3}>{t(($) => $.unsubscribed.title)}</Title>
            <Text c="dimmed" ta="center">
              {t(($) => $.unsubscribed.body)}
            </Text>
            {all ? (
              <Anchor href={all} size="sm" mt="md">
                {t(($) => $.unsubscribed.unsubscribeAll)}
              </Anchor>
            ) : null}
          </Stack>
        ) : (
          <Stack align="center" gap="md">
            <Title order={3}>{t(($) => $.unsubscribe.title)}</Title>
            <Text c="dimmed" ta="center">
              {t(($) => $.unsubscribe.body)}
            </Text>
            {status === 'error' ? (
              <Alert color="red" title={t(($) => $.unsubscribe.errorTitle)} w="100%">
                {t(($) => $.unsubscribe.errorBody)}
              </Alert>
            ) : null}
            <Button onClick={confirm} loading={status === 'pending'} color="red">
              {t(($) => $.unsubscribe.confirm)}
            </Button>
          </Stack>
        )}
      </Card>
    </Stack>
  )
}
