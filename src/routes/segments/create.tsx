import { Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import {
  siteSegmentsCreateMutation,
  siteSegmentsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import { segmentsCreateRoute, segmentsEditRoute } from '../../router.tsx'
import { SegmentForm, type SegmentFormValues } from './SegmentForm.tsx'

export function SegmentCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = segmentsCreateRoute.useParams()

  const form = useForm<SegmentFormValues>({
    initialValues: { name: '', definition: '' },
  })

  const createMutation = useResourceMutation({
    mutation: siteSegmentsCreateMutation(),
    invalidate: [siteSegmentsListQueryKey({ path: { slug: slug } })],
    successMessage: t(($) => $.notifications.segmentCreated),
    errorTitle: t(($) => $.alerts.segmentSaveErrorTitle),
    onDone: (created) =>
      navigate({ to: segmentsEditRoute.to, params: { slug, segmentId: created.id } }),
  })

  return (
    <Stack>
      <Title order={4}>{t(($) => $.segments.createTitle)}</Title>
      <SegmentForm
        form={form}
        isPending={createMutation.isPending}
        onSubmit={(values) =>
          createMutation.mutate({
            path: { slug: slug },
            body: {
              name: values.name.trim(),
              definition: values.definition.trim() || null,
            },
          })
        }
      />
    </Stack>
  )
}
