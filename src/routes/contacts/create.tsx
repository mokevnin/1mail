import { Button, Group, Stack, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { trpc } from '../../../trpc/client.ts'
import { EMPTY_CONTACT_FORM, toCreateNullableField } from './form.ts'

export function ContactCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const utils = trpc.useUtils()
  const form = useForm({
    initialValues: EMPTY_CONTACT_FORM,
  })

  const createContactMutation = trpc.contacts.create.useMutation({
    onSuccess: async (created) => {
      await utils.contacts.list.invalidate()
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.contactCreated),
      })
      await navigate({
        to: '/contacts/$contactId/edit',
        params: { contactId: created.id },
      })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.alerts.saveErrorTitle),
        message: error.message,
      })
    },
  })

  const onSubmit = form.onSubmit((values) => {
    createContactMutation.mutate({
      email: values.email.trim(),
      firstName: toCreateNullableField(values.firstName),
      lastName: toCreateNullableField(values.lastName),
      timeZone: toCreateNullableField(values.timeZone),
    })
  })

  return (
    <Stack>
      <Title order={4}>{t(($) => $.form.createTitle)}</Title>

      <form onSubmit={onSubmit}>
        <Stack>
          <TextInput label="Email" {...form.getInputProps('email')} required />
          <TextInput label={t(($) => $.table.firstName)} {...form.getInputProps('firstName')} />
          <TextInput label={t(($) => $.table.lastName)} {...form.getInputProps('lastName')} />
          <TextInput label={t(($) => $.table.timeZone)} {...form.getInputProps('timeZone')} />

          <Group justify="flex-end">
            <Button variant="default" type="button" onClick={() => navigate({ to: '/contacts' })}>
              {t(($) => $.actions.cancel)}
            </Button>
            <Button type="submit" loading={createContactMutation.isPending}>
              {t(($) => $.actions.save)}
            </Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  )
}
