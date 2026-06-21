import {
  Alert,
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
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  siteUserGetMeOptions,
  siteUserGetMeQueryKey,
  siteUserUpdateMeMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteUpdateMeInput, SiteUserResource } from '../../generated/site/types.gen.ts'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'

type ProfileFormValues = {
  name: string
  currentPassword: string
  newPassword: string
}

// ProfileForm is split out so the Mantine form seeds its initial values from the
// loaded user without an effect — the parent only renders it once data is ready.
function ProfileForm({ user }: { user: SiteUserResource }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const form = useForm<ProfileFormValues>({
    initialValues: { name: user.name, currentPassword: '', newPassword: '' },
  })

  const updateMutation = useMutation({
    ...siteUserUpdateMeMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: siteUserGetMeQueryKey() })
      form.setFieldValue('currentPassword', '')
      form.setFieldValue('newPassword', '')
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.profile.updated),
      })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.profile.errorTitle),
        message: getApiErrorMessage(
          error,
          t(($) => $.profile.errorTitle),
        ),
      })
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
        <TextInput label={t(($) => $.profile.emailLabel)} value={user.email} disabled />

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

      {meQuery.data ? <ProfileForm user={meQuery.data} /> : null}
    </Stack>
  )
}
