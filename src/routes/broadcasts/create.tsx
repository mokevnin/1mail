import { Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteBroadcastsCreateMutation,
  siteBroadcastsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { broadcastsCreateRoute, broadcastsEditRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import { BroadcastForm, type BroadcastFormValues } from './BroadcastForm.tsx'

export function BroadcastCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { slug } = broadcastsCreateRoute.useParams()

  const form = useForm<BroadcastFormValues>({
    initialValues: { name: '', subject: '', fromName: '', fromEmail: '', bodyHtml: '' },
  })

  const createMutation = useMutation({
    ...siteBroadcastsCreateMutation(),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({
        queryKey: siteBroadcastsListQueryKey({ path: { workspaceSlug: slug } }),
      })
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.broadcastCreated),
      })
      await navigate({ to: broadcastsEditRoute.to, params: { slug, broadcastId: created.id } })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.alerts.broadcastSaveErrorTitle),
        message: getApiErrorMessage(
          error,
          t(($) => $.alerts.broadcastSaveErrorTitle),
        ),
      })
    },
  })

  return (
    <Stack>
      <Title order={4}>{t(($) => $.broadcasts.createTitle)}</Title>
      <BroadcastForm
        form={form}
        isPending={createMutation.isPending}
        onSubmit={(values) =>
          createMutation.mutate({
            path: { workspaceSlug: slug },
            body: {
              name: values.name.trim(),
              subject: values.subject.trim(),
              fromName: values.fromName.trim() || null,
              fromEmail: values.fromEmail.trim() || null,
              bodyHtml: values.bodyHtml,
            },
          })
        }
      />
    </Stack>
  )
}
