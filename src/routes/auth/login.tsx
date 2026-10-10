import { Anchor, Button, Group, PasswordInput, Stack, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import { useNavigate, useRouter, useSearch } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { siteAuthLoginMutation } from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteLoginInput } from '../../generated/site/types.gen.ts'
import { useApiErrorMessage } from '../../hooks/useApiErrorMessage.ts'
import { forgotPasswordRoute, indexRoute, registerRoute } from '../../router.tsx'
import { getRateLimitWait } from '../../utils/apiErrors.ts'

export function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const router = useRouter()
  // Set when a guard (e.g. the OAuth consent screen) sent the user here to sign in.
  const { redirect } = useSearch({ strict: false })

  const apiErrorMessage = useApiErrorMessage()

  // How long to wait, when the throttle says so (ADR 0025).
  const rateLimitMessage = (error: unknown) => {
    const wait = getRateLimitWait(error)
    if (!wait) return undefined
    return wait.unit === 'minutes'
      ? t(($) => $.login.rateLimitedMinutes, { count: wait.count })
      : t(($) => $.login.rateLimitedSeconds, { count: wait.count })
  }

  const form = useForm<SiteLoginInput>({
    initialValues: {
      email: '',
      password: '',
    },
  })

  const loginMutation = useMutation({
    ...siteAuthLoginMutation(),
    onSuccess: () => (redirect ? router.history.push(redirect) : navigate({ to: indexRoute.to })),
    onError: (error) => {
      // Drop the rejected password so the user retypes a fresh one.
      form.setFieldValue('password', '')
      notifications.show({
        color: 'red',
        title: t(($) => $.login.errorTitle),
        message:
          rateLimitMessage(error) ??
          apiErrorMessage(
            error,
            t(($) => $.login.errorMessage),
          ),
      })
    },
  })

  const handleSubmit = (values: SiteLoginInput) => {
    loginMutation.mutate({
      body: {
        email: values.email.trim(),
        password: values.password,
      },
    })
  }

  return (
    <Stack maw={400} mx="auto" mt="xl">
      <Title order={3}>{t(($) => $.login.title)}</Title>

      <form onSubmit={form.onSubmit(handleSubmit)}>
        <Stack>
          <TextInput
            label={t(($) => $.login.emailLabel)}
            type="email"
            required
            {...form.getInputProps('email')}
          />
          <PasswordInput
            label={t(($) => $.login.passwordLabel)}
            required
            {...form.getInputProps('password')}
          />

          <Group justify="space-between" align="center">
            <Stack gap={2}>
              <Anchor component="a" href={registerRoute.to} size="sm">
                {t(($) => $.login.registerLink)}
              </Anchor>
              <Anchor component="a" href={forgotPasswordRoute.to} size="sm">
                {t(($) => $.login.forgotPasswordLink)}
              </Anchor>
            </Stack>
            <Button type="submit" loading={loginMutation.isPending}>
              {t(($) => $.login.submitButton)}
            </Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  )
}
