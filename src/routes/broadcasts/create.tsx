import { Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteBroadcastsCreateMutation,
  siteBroadcastsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import { broadcastsCreateRoute, broadcastsEditRoute } from '../../router.tsx'
import { BroadcastForm, type BroadcastFormValues } from './BroadcastForm.tsx'

export function BroadcastCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = broadcastsCreateRoute.useParams()

  const form = useForm<BroadcastFormValues>({
    initialValues: { name: '', subject: '', fromName: '', fromEmail: '', body: '', segmentId: '' },
  })

  const createMutation = useResourceMutation({
    mutation: siteBroadcastsCreateMutation(),
    invalidate: [siteBroadcastsListQueryKey({ path: { slug: slug } })],
    successMessage: t(($) => $.notifications.broadcastCreated),
    errorTitle: t(($) => $.alerts.broadcastSaveErrorTitle),
    onDone: (created) =>
      navigate({ to: broadcastsEditRoute.to, params: { slug, broadcastId: created.id } }),
  })

  return (
    <Stack>
      <Title order={4}>{t(($) => $.broadcasts.createTitle)}</Title>
      <BroadcastForm
        form={form}
        isPending={createMutation.isPending}
        onSubmit={(values) =>
          createMutation.mutate({
            path: { slug: slug },
            body: {
              name: values.name.trim(),
              subject: values.subject.trim(),
              fromName: values.fromName.trim() || null,
              fromEmail: values.fromEmail.trim() || null,
              body: values.body,
              segmentId: values.segmentId || null,
            },
          })
        }
      />
    </Stack>
  )
}
