import { Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteContactsCreateMutation,
  siteContactsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import { contactsCreateRoute, contactsEditRoute } from '../../router.tsx'
import { ContactForm, type ContactFormValues } from './ContactForm.tsx'

export function ContactCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = contactsCreateRoute.useParams()

  const form = useForm<ContactFormValues>({
    initialValues: { email: '', firstName: '', lastName: '', timeZone: '' },
  })

  const createMutation = useResourceMutation({
    mutation: siteContactsCreateMutation(),
    invalidate: [siteContactsListQueryKey({ path: { workspaceSlug: slug } })],
    successMessage: t(($) => $.notifications.contactCreated),
    errorTitle: t(($) => $.alerts.saveErrorTitle),
    onDone: (created) =>
      navigate({ to: contactsEditRoute.to, params: { slug, contactId: created.id } }),
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
