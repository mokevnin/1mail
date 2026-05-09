import { Button, Group, PasswordInput, Stack, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { siteAuthRegister } from '../../generated/site/sdk.gen.ts'
import { type SiteRegisterInput } from '../../generated/site/types.gen.ts'
import { contactsRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'

export function RegisterPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const registerMutation = useMutation({
    mutationFn: (body: SiteRegisterInput) => siteAuthRegister({ body }),
  })

  const form = useForm<SiteRegisterInput>({
    initialValues: {
      name: '',
      email: '',
      password: '',
    },
  })

  const handleSubmit = async (values: SiteRegisterInput) => {
    const { error } = await registerMutation.mutateAsync({
      name: values.name.trim(),
      email: values.email.trim(),
      password: values.password,
    })

    if (error) {
      notifications.show({
        color: 'red',
        title: t(($) => $.registration.errorTitle),
        message: getApiErrorMessage(error, t(($) => $.registration.errorTitle)),
      })
      return
    }

    notifications.show({
      color: 'teal',
      title: t(($) => $.notifications.successTitle),
      message: t(($) => $.registration.successMessage),
    })
    await navigate({ to: contactsRoute.to })
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
