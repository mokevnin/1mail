import { Alert, Loader, Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteContactsGetOptions,
  siteContactsGetQueryKey,
  siteContactsListQueryKey,
  siteContactsUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { contactsEditRoute } from '../../router.tsx'
import { track } from '../../tracking.ts'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import { ContactForm } from './ContactForm.tsx'

export function ContactEditPage() {
  const { t } = useTranslation()
  const { contactId } = contactsEditRoute.useParams()
  const queryClient = useQueryClient()

  const form = useForm({
    initialValues: { email: '', firstName: '', lastName: '', timeZone: '' },
  })

  const getContactQuery = useQuery(siteContactsGetOptions({ path: { id: contactId } }))

  useEffect(() => {
    if (!getContactQuery.data) return
    form.setValues({
      email: getContactQuery.data.email,
      firstName: getContactQuery.data.firstName ?? '',
      lastName: getContactQuery.data.lastName ?? '',
      timeZone: getContactQuery.data.timeZone ?? '',
    })
  }, [getContactQuery.data])

  const updateMutation = useMutation({
    ...siteContactsUpdateMutation(),
    onSuccess: async (updated) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: siteContactsListQueryKey() }),
        queryClient.invalidateQueries({ queryKey: siteContactsGetQueryKey({ path: { id: updated.id } }) }),
      ])
      await track('contact.updated', { contactId: updated.id, email: updated.email })
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
        message: getApiErrorMessage(error, t(($) => $.alerts.saveErrorTitle)),
      })
    },
  })

  if (getContactQuery.isLoading) return <Loader />

  if (getContactQuery.isError) {
    return (
      <ApiErrorAlert
        error={getContactQuery.error}
        title={t(($) => $.alerts.loadErrorTitle)}
        fallback={t(($) => $.alerts.loadErrorTitle)}
      />
    )
  }

  return (
    <Stack>
      <Title order={4}>{t(($) => $.form.editTitle)}</Title>
      <ContactForm
        form={form}
        emailEditable={false}
        isPending={updateMutation.isPending}
        onSubmit={({ firstName, lastName, timeZone }) =>
          updateMutation.mutate({ path: { id: contactId }, body: { firstName, lastName, timeZone } })
        }
      />
    </Stack>
  )
}
