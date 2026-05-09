import { Anchor, Button, Group, PasswordInput, Stack, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { siteAuthDirectLoginMutation } from '../../generated/site/@tanstack/react-query.gen.ts'
import { contactsRoute, registerRoute } from '../../router.tsx'

export function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const form = useForm({
    initialValues: {
      email: '',
      password: '',
    },
  })

  const loginMutation = useMutation({
    ...siteAuthDirectLoginMutation(),
    onSuccess: async () => {
      await navigate({ to: contactsRoute.to })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.login.errorTitle),
        message: 'error' in error ? error.error : (error.detail ?? error.title),
      })
    },
  })

  const onSubmit = form.onSubmit((values) => {
    loginMutation.mutate({
      body: {
        user: values.email.trim(),
        passwd: values.password,
      },
    })
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
            <Button type="submit" loading={loginMutation.isPending}>
              {t(($) => $.login.submitButton)}
            </Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  )
}
