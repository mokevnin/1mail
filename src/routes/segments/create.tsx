import { Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteSegmentsCreateMutation,
  siteSegmentsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { SiteSegmentType } from '../../generated/site/types.gen.ts'
import { segmentsCreateRoute, segmentsEditRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import { SegmentForm, type SegmentFormValues } from './SegmentForm.tsx'

export function SegmentCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { slug } = segmentsCreateRoute.useParams()

  const form = useForm<SegmentFormValues>({
    initialValues: { name: '', type: SiteSegmentType.RULE, definition: '' },
  })

  const createMutation = useMutation({
    ...siteSegmentsCreateMutation(),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({
        queryKey: siteSegmentsListQueryKey({ path: { workspaceSlug: slug } }),
      })
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.segmentCreated),
      })
      await navigate({ to: segmentsEditRoute.to, params: { slug, segmentId: created.id } })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.alerts.segmentSaveErrorTitle),
        message: getApiErrorMessage(
          error,
          t(($) => $.alerts.segmentSaveErrorTitle),
        ),
      })
    },
  })

  return (
    <Stack>
      <Title order={4}>{t(($) => $.segments.createTitle)}</Title>
      <SegmentForm
        form={form}
        isPending={createMutation.isPending}
        onSubmit={(values) =>
          createMutation.mutate({
            path: { workspaceSlug: slug },
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
