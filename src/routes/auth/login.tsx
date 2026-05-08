import { Anchor, Button, Group, PasswordInput, Stack, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { registerRoute } from '../../router.tsx'

export function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const form = useForm({
    initialValues: {
      email: '',
      password: '',
    },
  })

  const onSubmit = form.onSubmit(async (values) => {
    const params = new URLSearchParams({ user: values.email, passwd: values.password })
    const res = await fetch(`/auth/direct/login?${params}`, { credentials: 'include' })
    if (res.ok) {
      await navigate({ to: '/contacts' })
    } else {
      notifications.show({
        color: 'red',
        title: t(($) => $.login.errorTitle),
        message: res.statusText || String(res.status),
      })
    }
  })

  return (
    <Stack maw={400} mx="auto" mt="xl">
      <Title order={3}>{t(($) => $.login.title)}</Title>

      <form onSubmit={onSubmit}>
        <Stack>
          <TextInput
            label={t(($) => $.login.emailLabel)}
            type="email"
            {...form.getInputProps('email')}
            required
          />
          <PasswordInput
            label={t(($) => $.login.passwordLabel)}
            {...form.getInputProps('password')}
            required
          />

          <Group justify="space-between" align="center">
            <Anchor component="a" href={registerRoute.to} size="sm">
              {t(($) => $.login.registerLink)}
            </Anchor>
            <Button type="submit" loading={form.submitting}>
              {t(($) => $.login.submitButton)}
            </Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  )
}
