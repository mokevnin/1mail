import { Button, Group, PasswordInput, Stack, TextInput, Title } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useForm } from '@tanstack/react-form'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { siteAuthRegister } from '../../generated/site/sdk.gen.ts'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'

export function RegisterPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const { Field, Subscribe, handleSubmit } = useForm({
    defaultValues: {
      name: '',
      email: '',
      password: '',
    },
    validators: {
      onSubmitAsync: async ({ value }) => {
        const { error } = await siteAuthRegister({
          body: {
            name: value.name.trim(),
            email: value.email.trim(),
            password: value.password,
          },
        })

        if (error) {
          notifications.show({
            color: 'red',
            title: t(($) => $.registration.errorTitle),
            message: getApiErrorMessage(
              error,
              t(($) => $.registration.errorTitle),
            ),
          })
          return error
        }

        notifications.show({
          color: 'teal',
          title: t(($) => $.notifications.successTitle),
          message: t(($) => $.registration.successMessage),
        })
        await navigate({ to: '/contacts' })
      },
    },
  })

  return (
    <Stack maw={400} mx="auto" mt="xl">
      <Title order={3}>{t(($) => $.registration.title)}</Title>

      <form
        onSubmit={(event) => {
          event.preventDefault()
          void handleSubmit()
        }}
      >
        <Stack>
          <Field name="name">
            {({ state, handleChange, handleBlur }) => (
              <TextInput
                label={t(($) => $.registration.nameLabel)}
                value={state.value}
                onBlur={handleBlur}
                onChange={(event) => handleChange(event.currentTarget.value)}
                error={state.meta.errors.join(', ') || undefined}
                required
              />
            )}
          </Field>
          <Field name="email">
            {({ state, handleChange, handleBlur }) => (
              <TextInput
                label={t(($) => $.registration.emailLabel)}
                type="email"
                value={state.value}
                onBlur={handleBlur}
                onChange={(event) => handleChange(event.currentTarget.value)}
                error={state.meta.errors.join(', ') || undefined}
                required
              />
            )}
          </Field>
          <Field name="password">
            {({ state, handleChange, handleBlur }) => (
              <PasswordInput
                label={t(($) => $.registration.passwordLabel)}
                value={state.value}
                onBlur={handleBlur}
                onChange={(event) => handleChange(event.currentTarget.value)}
                error={state.meta.errors.join(', ') || undefined}
                required
              />
            )}
          </Field>

          <Group justify="flex-end">
            <Subscribe selector={(state) => state.isSubmitting}>
              {(isSubmitting) => (
                <Button type="submit" loading={isSubmitting}>
                  {t(($) => $.registration.submitButton)}
                </Button>
              )}
            </Subscribe>
          </Group>
        </Stack>
      </form>
    </Stack>
  )
}
