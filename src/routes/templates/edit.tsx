import { Loader, Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useEffectEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteTemplatesGetOptions,
  siteTemplatesGetQueryKey,
  siteTemplatesListQueryKey,
  siteTemplatesUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteEmailTemplateResource } from '../../generated/site/types.gen.ts'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import { templatesEditRoute } from '../../router.tsx'
import { TemplateForm, type TemplateFormValues } from './TemplateForm.tsx'

export function TemplateEditPage() {
  const { t } = useTranslation()
  const { slug, templateId } = templatesEditRoute.useParams()

  const form = useForm<TemplateFormValues>({
    initialValues: { name: '', subject: '', body: '' },
  })

  const getQuery = useQuery(siteTemplatesGetOptions({ path: { slug: slug, id: templateId } }))

  const applyData = useEffectEvent((data: SiteEmailTemplateResource | undefined) => {
    if (!data) return
    form.setValues({
      name: data.name,
      subject: data.subject,
      body: data.body,
    })
  })

  useEffect(() => {
    applyData(getQuery.data)
  }, [getQuery.data])

  const updateMutation = useResourceMutation({
    mutation: siteTemplatesUpdateMutation(),
    invalidate: [
      siteTemplatesListQueryKey({ path: { slug: slug } }),
      siteTemplatesGetQueryKey({ path: { slug: slug, id: templateId } }),
    ],
    successMessage: t(($) => $.notifications.templateUpdated),
    errorTitle: t(($) => $.alerts.templateSaveErrorTitle),
  })

  if (getQuery.isLoading) return <Loader />

  if (getQuery.isError) {
    return (
      <ApiErrorAlert
        error={getQuery.error}
        title={t(($) => $.alerts.templateLoadErrorTitle)}
        fallback={t(($) => $.alerts.templateLoadErrorTitle)}
      />
    )
  }

  return (
    <Stack>
      <Title order={4}>{t(($) => $.templates.editTitle)}</Title>
      <TemplateForm
        form={form}
        isPending={updateMutation.isPending}
        onSubmit={(values) =>
          updateMutation.mutate({
            path: { slug: slug, id: templateId },
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
