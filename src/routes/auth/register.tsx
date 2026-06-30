import { Button, Group, PasswordInput, Stack, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { siteAuthRegisterMutation } from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteRegisterInput } from '../../generated/site/types.gen.ts'
import { indexRoute } from '../../router.tsx'
import { type ApiErrorLike, getApiErrorMessage } from '../../utils/apiErrors.ts'

export function RegisterPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const registerMutation = useMutation(siteAuthRegisterMutation())

  const form = useForm<SiteRegisterInput>({
    initialValues: {
      name: '',
      email: '',
      password: '',
    },
  })

  const handleSubmit = async (values: SiteRegisterInput) => {
    try {
      await registerMutation.mutateAsync({
        body: {
          name: values.name.trim(),
          email: values.email.trim(),
          password: values.password,
        },
      })
    } catch (error) {
      form.setFieldValue('password', '')
      notifications.show({
        color: 'red',
        title: t(($) => $.registration.errorTitle),
        message: getApiErrorMessage(
          error as ApiErrorLike,
          t(($) => $.registration.errorTitle),
        ),
      })
      return
    }

    notifications.show({
      color: 'teal',
      title: t(($) => $.notifications.successTitle),
      message: t(($) => $.registration.successMessage),
    })
    await navigate({ to: indexRoute.to })
  }

  return (
    <Stack maw={400} mx="auto" mt="xl">
      <Title order={3}>{t(($) => $.registration.title)}</Title>

      <form onSubmit={form.onSubmit(handleSubmit)}>
        <Stack>
          <TextInput
            label={t(($) => $.registration.nameLabel)}
            required
            {...form.getInputProps('name')}
          />
          <TextInput
            label={t(($) => $.registration.emailLabel)}
            type="email"
            required
            {...form.getInputProps('email')}
          />
          <PasswordInput
            label={t(($) => $.registration.passwordLabel)}
            required
            {...form.getInputProps('password')}
          />

          <Group justify="flex-end">
            <Button type="submit" loading={registerMutation.isPending}>
              {t(($) => $.registration.submitButton)}
            </Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  )
}
