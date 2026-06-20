import { track } from '@1mail/analytics'
import { Loader, Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useEffectEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteContactsGetOptions,
  siteContactsGetQueryKey,
  siteContactsListQueryKey,
  siteContactsUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteContactResource } from '../../generated/site/types.gen.ts'
import { contactsEditRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import { ContactForm, type ContactFormValues } from './ContactForm.tsx'

export function ContactEditPage() {
  const { t } = useTranslation()
  const { slug, contactId } = contactsEditRoute.useParams()
  const queryClient = useQueryClient()

  const form = useForm<ContactFormValues>({
    initialValues: { email: '', firstName: '', lastName: '', timeZone: '' },
  })

  const getContactQuery = useQuery(
    siteContactsGetOptions({ path: { workspaceSlug: slug, id: contactId } }),
  )

  const applyContactData = useEffectEvent((data: SiteContactResource | undefined) => {
    if (!data) return
    form.setValues({
      email: data.email,
      firstName: data.firstName ?? '',
      lastName: data.lastName ?? '',
      timeZone: data.timeZone ?? '',
    })
  })

  useEffect(() => {
    applyContactData(getContactQuery.data)
  }, [getContactQuery.data])

  const updateMutation = useMutation({
    ...siteContactsUpdateMutation(),
    onSuccess: async (updated) => {
      await queryClient.invalidateQueries({
        queryKey: siteContactsListQueryKey({ path: { workspaceSlug: slug } }),
      })
      await queryClient.invalidateQueries({
        queryKey: siteContactsGetQueryKey({ path: { workspaceSlug: slug, id: updated.id } }),
      })
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
        message: getApiErrorMessage(
          error,
          t(($) => $.alerts.saveErrorTitle),
        ),
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
          updateMutation.mutate({
            path: { workspaceSlug: slug, id: contactId },
            body: { firstName, lastName, timeZone },
          })
        }
      />
    </Stack>
  )
}
