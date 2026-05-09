import { Button, Group, PasswordInput, Stack, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { siteAuthRegisterMutation } from '../../generated/site/@tanstack/react-query.gen.ts'
import { useApiFormErrors } from '../../hooks/use-api-form-errors.ts'

export function RegisterPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const form = useForm({
    initialValues: {
      name: '',
      email: '',
      password: '',
    },
  })
  const withFormErrors = useApiFormErrors(form)

  const registerMutation = useMutation(
    withFormErrors({
      ...siteAuthRegisterMutation(),
      onSuccess: async () => {
        notifications.show({
          color: 'teal',
          title: t(($) => $.notifications.successTitle),
          message: t(($) => $.registration.successMessage),
        })
        await navigate({ to: '/contacts' })
      },
      onError: (error) => {
        notifications.show({
          color: 'red',
          title: t(($) => $.registration.errorTitle),
          message: error.detail ?? error.title,
        })
      },
    }),
  )

  const onSubmit = form.onSubmit((values) => {
    registerMutation.mutate({
      body: {
        name: values.name.trim(),
        email: values.email.trim(),
        password: values.password,
      },
    })
  })

  return (
    <Stack maw={400} mx="auto" mt="xl">
      <Title order={3}>{t(($) => $.registration.title)}</Title>

      <form onSubmit={onSubmit}>
        <Stack>
          <TextInput
            label={t(($) => $.registration.nameLabel)}
            {...form.getInputProps('name')}
            required
          />
          <TextInput
            label={t(($) => $.registration.emailLabel)}
            type="email"
            {...form.getInputProps('email')}
            required
          />
          <PasswordInput
            label={t(($) => $.registration.passwordLabel)}
            {...form.getInputProps('password')}
            required
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
