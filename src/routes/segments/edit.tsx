import { Loader, Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useEffectEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteSegmentsGetOptions,
  siteSegmentsGetQueryKey,
  siteSegmentsListQueryKey,
  siteSegmentsUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { type SiteSegmentResource, SiteSegmentType } from '../../generated/site/types.gen.ts'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import { segmentsEditRoute } from '../../router.tsx'
import { SegmentForm, type SegmentFormValues } from './SegmentForm.tsx'

export function SegmentEditPage() {
  const { t } = useTranslation()
  const { slug, segmentId } = segmentsEditRoute.useParams()

  const form = useForm<SegmentFormValues>({
    initialValues: { name: '', type: SiteSegmentType.RULE, definition: '' },
  })

  const getSegmentQuery = useQuery(siteSegmentsGetOptions({ path: { slug: slug, id: segmentId } }))

  const applySegmentData = useEffectEvent((data: SiteSegmentResource | undefined) => {
    if (!data) return
    form.setValues({
      name: data.name,
      type: data.type,
      definition: data.definition ?? '',
    })
  })

  useEffect(() => {
    applySegmentData(getSegmentQuery.data)
  }, [getSegmentQuery.data])

  const updateMutation = useResourceMutation({
    mutation: siteSegmentsUpdateMutation(),
    invalidate: [
      siteSegmentsListQueryKey({ path: { slug: slug } }),
      siteSegmentsGetQueryKey({ path: { slug: slug, id: segmentId } }),
    ],
    successMessage: t(($) => $.notifications.segmentUpdated),
    errorTitle: t(($) => $.alerts.segmentSaveErrorTitle),
  })

  if (getSegmentQuery.isLoading) return <Loader />

  if (getSegmentQuery.isError) {
    return (
      <ApiErrorAlert
        error={getSegmentQuery.error}
        title={t(($) => $.alerts.segmentLoadErrorTitle)}
        fallback={t(($) => $.alerts.segmentLoadErrorTitle)}
      />
    )
  }

  return (
    <Stack>
      <Title order={4}>{t(($) => $.segments.editTitle)}</Title>
      <SegmentForm
        form={form}
        isPending={updateMutation.isPending}
        onSubmit={(values) =>
          updateMutation.mutate({
            path: { slug: slug, id: segmentId },
            body: {
              name: values.name.trim(),
              type: values.type,
              definition: values.definition.trim() || null,
            },
          })
        }
      />
    </Stack>
  )
}
