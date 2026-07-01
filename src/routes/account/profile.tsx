import {
  Alert,
  Badge,
  Button,
  Divider,
  Group,
  Loader,
  PasswordInput,
  Stack,
  TextInput,
  Title,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  siteUserEmailChangeMutation,
  siteUserGetMeOptions,
  siteUserGetMeQueryKey,
  siteUserResendVerificationMutation,
  siteUserUpdateMeMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type {
  SiteEmailChangeInput,
  SiteUpdateMeInput,
  SiteUserResource,
} from '../../generated/site/types.gen.ts'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'

type ProfileFormValues = {
  name: string
  currentPassword: string
  newPassword: string
}

// ProfileForm is split out so the Mantine form seeds its initial values from the
// loaded user without an effect — the parent only renders it once data is ready.
function ProfileForm({ user }: { user: SiteUserResource }) {
  const { t } = useTranslation()

  const form = useForm<ProfileFormValues>({
    initialValues: { name: user.name, currentPassword: '', newPassword: '' },
  })

  const updateMutation = useResourceMutation({
    mutation: siteUserUpdateMeMutation(),
    invalidate: [siteUserGetMeQueryKey()],
    successMessage: t(($) => $.profile.updated),
    errorTitle: t(($) => $.profile.errorTitle),
    onDone: () => {
      form.setFieldValue('currentPassword', '')
      form.setFieldValue('newPassword', '')
    },
  })

  const handleSubmit = (values: ProfileFormValues) => {
    const body: SiteUpdateMeInput = { name: values.name.trim() }
    if (values.newPassword) {
      body.currentPassword = values.currentPassword
      body.newPassword = values.newPassword
    }
    updateMutation.mutate({ body })
  }

  return (
    <form onSubmit={form.onSubmit(handleSubmit)}>
      <Stack maw={420}>
        <TextInput label={t(($) => $.profile.nameLabel)} required {...form.getInputProps('name')} />
        <TextInput
          label={t(($) => $.profile.emailLabel)}
          value={user.email}
          disabled
          rightSectionWidth={110}
          rightSection={
            <Badge color={user.emailVerified ? 'teal' : 'gray'} variant="light">
              {user.emailVerified
                ? t(($) => $.profile.verifiedBadge)
                : t(($) => $.profile.unverifiedBadge)}
            </Badge>
          }
        />

        <Divider label={t(($) => $.profile.passwordSectionTitle)} />
        <PasswordInput
          label={t(($) => $.profile.currentPasswordLabel)}
          {...form.getInputProps('currentPassword')}
        />
        <PasswordInput
          label={t(($) => $.profile.newPasswordLabel)}
          {...form.getInputProps('newPassword')}
        />

        <Group justify="flex-end">
          <Button type="submit" loading={updateMutation.isPending}>
            {t(($) => $.actions.save)}
          </Button>
        </Group>
      </Stack>
    </form>
  )
}

// UnverifiedAlert prompts an unverified user to resend the verification email.
function UnverifiedAlert() {
  const { t } = useTranslation()
  const resend = useResourceMutation({
    mutation: siteUserResendVerificationMutation(),
    successMessage: t(($) => $.profile.resendSuccess),
    errorTitle: t(($) => $.profile.resendErrorTitle),
  })

  return (
    <Alert color="yellow" title={t(($) => $.profile.unverifiedAlert)}>
      <Button
        variant="light"
        size="xs"
        loading={resend.isPending}
        onClick={() => resend.mutate({})}
      >
        {t(($) => $.profile.resendVerification)}
      </Button>
    </Alert>
  )
}

// EmailSection requests an email change; the new address must confirm via link.
function EmailSection() {
  const { t } = useTranslation()
  const form = useForm({ initialValues: { newEmail: '', currentPassword: '' } })

  const changeMutation = useResourceMutation({
    mutation: siteUserEmailChangeMutation(),
    successMessage: t(($) => $.profile.changeEmailSuccess),
    errorTitle: t(($) => $.profile.changeEmailErrorTitle),
    onDone: () => form.reset(),
  })

  const handleSubmit = (values: SiteEmailChangeInput) => {
    changeMutation.mutate({
      body: { newEmail: values.newEmail.trim(), currentPassword: values.currentPassword },
    })
  }

  return (
    <form onSubmit={form.onSubmit(handleSubmit)}>
      <Stack maw={420}>
        <Divider label={t(($) => $.profile.emailSectionTitle)} />
        <TextInput
          label={t(($) => $.profile.newEmailLabel)}
          type="email"
          required
          {...form.getInputProps('newEmail')}
        />
        <PasswordInput
          label={t(($) => $.profile.currentPasswordForEmailLabel)}
          required
          {...form.getInputProps('currentPassword')}
        />
        <Group justify="flex-end">
          <Button type="submit" loading={changeMutation.isPending}>
            {t(($) => $.profile.changeEmailButton)}
          </Button>
        </Group>
      </Stack>
    </form>
  )
}

export function ProfilePage() {
  const { t } = useTranslation()
  const meQuery = useQuery(siteUserGetMeOptions())

  return (
    <Stack>
      <Title order={2}>{t(($) => $.profile.title)}</Title>

      {meQuery.isError ? (
        <Alert color="red" title={t(($) => $.profile.loadErrorTitle)}>
          {t(($) => $.profile.loadErrorTitle)}
        </Alert>
      ) : null}

      {meQuery.isLoading ? <Loader /> : null}

      {meQuery.data ? (
        <Stack>
          {meQuery.data.emailVerified ? null : <UnverifiedAlert />}
          <ProfileForm user={meQuery.data} />
          <EmailSection />
        </Stack>
      ) : null}
    </Stack>
  )
}
