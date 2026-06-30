import { Anchor, Button, Group, PasswordInput, Stack, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { siteAuthDirectLoginMutation } from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteDirectLoginInput } from '../../generated/site/types.gen.ts'
import { indexRoute, registerRoute } from '../../router.tsx'
import { type ApiErrorLike, getApiErrorMessage } from '../../utils/apiErrors.ts'

export function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const loginMutation = useMutation(siteAuthDirectLoginMutation())

  const form = useForm<SiteDirectLoginInput>({
    initialValues: {
      user: '',
      passwd: '',
    },
  })

  const handleSubmit = async (values: SiteDirectLoginInput) => {
    try {
      await loginMutation.mutateAsync({
        body: {
          user: values.user.trim(),
          passwd: values.passwd,
        },
      })
      await navigate({ to: indexRoute.to })
    } catch (error) {
      form.setFieldValue('passwd', '')
      notifications.show({
        color: 'red',
        title: t(($) => $.login.errorTitle),
        message: getApiErrorMessage(
          error as ApiErrorLike,
          t(($) => $.login.errorTitle),
        ),
      })
    }
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
            {...form.getInputProps('user')}
          />
          <PasswordInput
            label={t(($) => $.login.passwordLabel)}
            required
            {...form.getInputProps('passwd')}
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
