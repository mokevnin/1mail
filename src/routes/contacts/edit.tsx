import { Alert, Button, Group, Loader, Stack, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { track } from '../../tracking.ts'
import { trpc } from '../../trpc.ts'
import { EMPTY_CONTACT_FORM, toUpdateNullableField } from './form.ts'

function parseContactId(contactId: string): string | null {
  return /^\d+$/.test(contactId) ? contactId : null
}

export function ContactEditPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { contactId } = useParams({ from: '/contacts/$contactId/edit' })
  const parsedContactId = parseContactId(contactId)
  const utils = trpc.useUtils()
  const form = useForm({
    initialValues: EMPTY_CONTACT_FORM,
  })

  const getContactQuery = trpc.contacts.get.useQuery(
    { id: parsedContactId ?? '0' },
    { enabled: parsedContactId !== null },
  )

  useEffect(() => {
    if (!getContactQuery.data) {
      return
    }

    form.setValues({
      email: getContactQuery.data.email,
      firstName: getContactQuery.data.firstName ?? '',
      lastName: getContactQuery.data.lastName ?? '',
      timeZone: getContactQuery.data.timeZone ?? '',
    })
  }, [form, getContactQuery.data])

  const updateContactMutation = trpc.contacts.update.useMutation({
    onSuccess: async (updated) => {
      await Promise.all([
        utils.contacts.list.invalidate(),
        utils.contacts.get.invalidate({ id: updated.id }),
      ])
      await track('contact.updated', {
        contactId: updated.id,
        email: updated.email,
      })

      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.contactUpdated),
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
    if (parsedContactId === null) {
      return
    }

    updateContactMutation.mutate({
      id: parsedContactId,
      data: {
        firstName: toUpdateNullableField(values.firstName),
        lastName: toUpdateNullableField(values.lastName),
        timeZone: toUpdateNullableField(values.timeZone),
      },
    })
  })

  if (parsedContactId === null) {
    return (
      <Alert color="red" title={t(($) => $.alerts.loadErrorTitle)}>
        {t(($) => $.form.invalidId)}
      </Alert>
    )
  }

  if (getContactQuery.isLoading) {
    return <Loader />
  }

  if (getContactQuery.isError) {
    return (
      <Alert color="red" title={t(($) => $.alerts.loadErrorTitle)}>
        {getContactQuery.error.message}
      </Alert>
    )
  }

  return (
    <Stack>
      <Title order={4}>{t(($) => $.form.editTitle)}</Title>

      <form onSubmit={onSubmit}>
        <Stack>
          <TextInput label="Email" {...form.getInputProps('email')} disabled />
          <TextInput label={t(($) => $.table.firstName)} {...form.getInputProps('firstName')} />
          <TextInput label={t(($) => $.table.lastName)} {...form.getInputProps('lastName')} />
          <TextInput label={t(($) => $.table.timeZone)} {...form.getInputProps('timeZone')} />

          <Group justify="flex-end">
            <Button variant="default" type="button" onClick={() => navigate({ to: '/contacts' })}>
              {t(($) => $.actions.cancel)}
            </Button>
            <Button type="submit" loading={updateContactMutation.isPending}>
              {t(($) => $.actions.save)}
            </Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  )
}
