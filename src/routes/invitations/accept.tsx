import { Alert, Button, Loader, PasswordInput, Stack, Text, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  sitePublicInvitationsAcceptMutation,
  sitePublicInvitationsLookupOptions,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { acceptInvitationRoute, loginRoute } from '../../router.tsx'
import { type ApiErrorLike, getApiErrorMessage } from '../../utils/apiErrors.ts'

export function AcceptInvitationPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { token } = acceptInvitationRoute.useParams()

  const lookup = useQuery({
    ...sitePublicInvitationsLookupOptions({ path: { token } }),
    retry: false,
  })

  const form = useForm({ initialValues: { name: '', password: '' } })

  const mutation = useMutation({
    ...sitePublicInvitationsAcceptMutation(),
    onSuccess: () => {
      notifications.show({
        color: 'teal',
        title: t(($) => $.acceptInvitation.successTitle),
        message: t(($) => $.acceptInvitation.successMessage),
      })
      navigate({ to: loginRoute.to })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.acceptInvitation.errorTitle),
        message: getApiErrorMessage(
          error as ApiErrorLike,
          t(($) => $.acceptInvitation.errorMessage),
        ),
      })
    },
  })

  if (lookup.isLoading) {
    return (
      <Stack maw={420} mx="auto" mt="xl" align="center">
        <Loader />
        <Text c="dimmed">{t(($) => $.acceptInvitation.loading)}</Text>
      </Stack>
    )
  }

  if (lookup.isError || !lookup.data) {
    return (
      <Stack maw={420} mx="auto" mt="xl">
        <Alert color="red" title={t(($) => $.acceptInvitation.errorTitle)}>
          {t(($) => $.acceptInvitation.invalid)}
        </Alert>
      </Stack>
    )
  }

  const invite = lookup.data
  const needsAccount = !invite.hasAccount

  const handleSubmit = (values: { name: string; password: string }) => {
    mutation.mutate({
      path: { token },
      body: needsAccount ? { name: values.name.trim(), password: values.password } : {},
    })
  }

  return (
    <Stack maw={420} mx="auto" mt="xl">
      <Title order={3}>{t(($) => $.acceptInvitation.title)}</Title>
      <Text>{t(($) => $.acceptInvitation.intro, { workspace: invite.workspaceName })}</Text>
      <Text c="dimmed" size="sm">
        {invite.email}
      </Text>

      <form onSubmit={form.onSubmit(handleSubmit)}>
        <Stack>
          {needsAccount ? (
            <>
              <TextInput
                label={t(($) => $.acceptInvitation.nameLabel)}
                required
                {...form.getInputProps('name')}
              />
              <PasswordInput
                label={t(($) => $.acceptInvitation.passwordLabel)}
                required
                {...form.getInputProps('password')}
              />
            </>
          ) : null}
          <Button type="submit" loading={mutation.isPending}>
            {needsAccount
              ? t(($) => $.acceptInvitation.acceptButton)
              : t(($) => $.acceptInvitation.acceptExistingButton)}
          </Button>
        </Stack>
      </form>
    </Stack>
  )
}
