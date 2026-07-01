import { Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteTemplatesCreateMutation,
  siteTemplatesListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import { templatesCreateRoute, templatesEditRoute } from '../../router.tsx'
import { TemplateForm, type TemplateFormValues } from './TemplateForm.tsx'

export function TemplateCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = templatesCreateRoute.useParams()

  const form = useForm<TemplateFormValues>({
    initialValues: { name: '', subject: '', body: '' },
  })

  const createMutation = useResourceMutation({
    mutation: siteTemplatesCreateMutation(),
    invalidate: [siteTemplatesListQueryKey({ path: { slug: slug } })],
    successMessage: t(($) => $.notifications.templateCreated),
    errorTitle: t(($) => $.alerts.templateSaveErrorTitle),
    onDone: (created) =>
      navigate({ to: templatesEditRoute.to, params: { slug, templateId: created.id } }),
  })

  return (
    <Stack>
      <Title order={4}>{t(($) => $.templates.createTitle)}</Title>
      <TemplateForm
        form={form}
        isPending={createMutation.isPending}
        onSubmit={(values) =>
          createMutation.mutate({
            path: { slug: slug },
            body: {
              name: values.name.trim(),
              subject: values.subject.trim(),
              body: values.body,
            },
          })
        }
      />
    </Stack>
  )
}
