import { Loader, Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
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
import { templatesEditRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import { TemplateForm, type TemplateFormValues } from './TemplateForm.tsx'

export function TemplateEditPage() {
  const { t } = useTranslation()
  const { slug, templateId } = templatesEditRoute.useParams()
  const queryClient = useQueryClient()

  const form = useForm<TemplateFormValues>({
    initialValues: { name: '', subject: '', body: '' },
  })

  const getQuery = useQuery(
    siteTemplatesGetOptions({ path: { workspaceSlug: slug, id: templateId } }),
  )

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

  const updateMutation = useMutation({
    ...siteTemplatesUpdateMutation(),
    onSuccess: async (updated) => {
      await queryClient.invalidateQueries({
        queryKey: siteTemplatesListQueryKey({ path: { workspaceSlug: slug } }),
      })
      await queryClient.invalidateQueries({
        queryKey: siteTemplatesGetQueryKey({ path: { workspaceSlug: slug, id: updated.id } }),
      })
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.templateUpdated),
      })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.alerts.templateSaveErrorTitle),
        message: getApiErrorMessage(
          error,
          t(($) => $.alerts.templateSaveErrorTitle),
        ),
      })
    },
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
            path: { workspaceSlug: slug, id: templateId },
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
