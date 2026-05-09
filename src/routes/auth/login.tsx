import { Anchor, Button, Group, PasswordInput, Stack, TextInput, Title } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useForm } from '@tanstack/react-form'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { siteAuthDirectLoginMutation } from '../../generated/site/@tanstack/react-query.gen.ts'
import { contactsRoute, registerRoute } from '../../router.tsx'
import { type ApiErrorLike, getApiErrorMessage } from '../../utils/apiErrors.ts'

export function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const loginMutation = useMutation(siteAuthDirectLoginMutation())

  const { Field, Subscribe, handleSubmit } = useForm({
    defaultValues: {
      email: '',
      password: '',
    },
    validators: {
      onSubmitAsync: async ({ value }) => {
        try {
          await loginMutation.mutateAsync({
            body: {
              user: value.email.trim(),
              passwd: value.password,
            },
          })
          await navigate({ to: contactsRoute.to })
        } catch (error) {
          notifications.show({
            color: 'red',
            title: t(($) => $.login.errorTitle),
            message: getApiErrorMessage(
              error as ApiErrorLike,
              t(($) => $.login.errorTitle),
            ),
          })
        }
      },
    },
  })

  return (
    <Stack maw={400} mx="auto" mt="xl">
      <Title order={3}>{t(($) => $.login.title)}</Title>

      <form
        onSubmit={(event) => {
          event.preventDefault()
          void handleSubmit()
        }}
      >
        <Stack>
          <Field name="email">
            {({ state, handleChange, handleBlur }) => (
              <TextInput
                label={t(($) => $.login.emailLabel)}
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
                label={t(($) => $.login.passwordLabel)}
                value={state.value}
                onBlur={handleBlur}
                onChange={(event) => handleChange(event.currentTarget.value)}
                error={state.meta.errors.join(', ') || undefined}
                required
              />
            )}
          </Field>

          <Group justify="space-between" align="center">
            <Anchor component="a" href={registerRoute.to} size="sm">
              {t(($) => $.login.registerLink)}
            </Anchor>
            <Subscribe selector={(state) => state.isSubmitting}>
              {(isSubmitting) => (
                <Button type="submit" loading={isSubmitting}>
                  {t(($) => $.login.submitButton)}
                </Button>
              )}
            </Subscribe>
          </Group>
        </Stack>
      </form>
    </Stack>
  )
}
