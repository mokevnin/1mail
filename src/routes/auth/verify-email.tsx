import { Anchor, Card, Loader, Stack, Text, Title } from '@mantine/core'
import { useMutation } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { siteAuthVerifyEmailMutation } from '../../generated/site/@tanstack/react-query.gen.ts'
import { indexRoute, verifyEmailRoute } from '../../router.tsx'

export function VerifyEmailPage() {
  const { t } = useTranslation()
  const { token } = verifyEmailRoute.useSearch()
  const mutation = useMutation(siteAuthVerifyEmailMutation())

  // Fire once on mount; the ref guards against StrictMode's double-invoke.
  const fired = useRef(false)
  useEffect(() => {
    if (fired.current) return
    fired.current = true
    mutation.mutate({ body: { token } })
  }, [mutation.mutate, token])

  return (
    <Stack maw={460} mx="auto" mt="xl" align="center">
      <Card withBorder w="100%" padding="xl">
        <Stack align="center" gap="sm">
          {mutation.isSuccess ? (
            <>
              <Title order={3}>{t(($) => $.verifyEmail.successTitle)}</Title>
              <Text c="dimmed" ta="center">
                {t(($) => $.verifyEmail.successBody)}
              </Text>
              <Anchor component="a" href={indexRoute.to} size="sm" mt="md">
                {t(($) => $.verifyEmail.continue)}
              </Anchor>
            </>
          ) : mutation.isError ? (
            <>
              <Title order={3}>{t(($) => $.verifyEmail.errorTitle)}</Title>
              <Text c="dimmed" ta="center">
                {t(($) => $.verifyEmail.errorBody)}
              </Text>
            </>
          ) : (
            <>
              <Loader />
              <Text c="dimmed">{t(($) => $.verifyEmail.verifying)}</Text>
            </>
          )}
        </Stack>
      </Card>
    </Stack>
  )
}
