import { track } from '@1mail/analytics'
import { Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteContactsCreateMutation,
  siteContactsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { contactsCreateRoute, contactsEditRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import { ContactForm, type ContactFormValues } from './ContactForm.tsx'

export function ContactCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { slug } = contactsCreateRoute.useParams()

  const form = useForm<ContactFormValues>({
    initialValues: { email: '', firstName: '', lastName: '', timeZone: '' },
  })

  const createMutation = useMutation({
    ...siteContactsCreateMutation(),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({
        queryKey: siteContactsListQueryKey({ path: { workspaceSlug: slug } }),
      })
      await track('contact.created', { contactId: created.id, email: created.email })
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.contactCreated),
      })
      await navigate({ to: contactsEditRoute.to, params: { slug, contactId: created.id } })
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

  return (
    <Stack>
      <Title order={4}>{t(($) => $.form.createTitle)}</Title>
      <ContactForm
        form={form}
        isPending={createMutation.isPending}
        onSubmit={(values) =>
          createMutation.mutate({
            path: { workspaceSlug: slug },
            body: { ...values, email: values.email.trim() },
          })
        }
      />
    </Stack>
  )
}
