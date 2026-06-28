import { Button, Divider, Group, Loader, Stack, Text, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useEffectEvent, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteBroadcastsGetOptions,
  siteBroadcastsGetQueryKey,
  siteBroadcastsListQueryKey,
  siteBroadcastsScheduleMutation,
  siteBroadcastsSendMutation,
  siteBroadcastsUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteBroadcastResource } from '../../generated/site/types.gen.ts'
import { broadcastsEditRoute, broadcastsReportRoute } from '../../router.tsx'
import { type ApiErrorLike, getApiErrorMessage } from '../../utils/apiErrors.ts'
import { BroadcastForm, type BroadcastFormValues } from './BroadcastForm.tsx'

export function BroadcastEditPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug, broadcastId } = broadcastsEditRoute.useParams()
  const queryClient = useQueryClient()
  const [scheduledAt, setScheduledAt] = useState('')

  const form = useForm<BroadcastFormValues>({
    initialValues: { name: '', subject: '', fromName: '', fromEmail: '', bodyHtml: '' },
  })

  const getQuery = useQuery(
    siteBroadcastsGetOptions({ path: { workspaceSlug: slug, id: broadcastId } }),
  )

  const applyData = useEffectEvent((data: SiteBroadcastResource | undefined) => {
    if (!data) return
    form.setValues({
      name: data.name,
      subject: data.subject,
      fromName: data.fromName ?? '',
      fromEmail: data.fromEmail ?? '',
      bodyHtml: data.bodyHtml,
    })
  })

  useEffect(() => {
    applyData(getQuery.data)
  }, [getQuery.data])

  const invalidate = async () => {
    await queryClient.invalidateQueries({
      queryKey: siteBroadcastsListQueryKey({ path: { workspaceSlug: slug } }),
    })
    await queryClient.invalidateQueries({
      queryKey: siteBroadcastsGetQueryKey({ path: { workspaceSlug: slug, id: broadcastId } }),
    })
  }

  const notifyError = (
    error: ApiErrorLike | null | undefined,
    titleKey: 'broadcastSaveErrorTitle' | 'broadcastSendErrorTitle',
  ) => {
    notifications.show({
      color: 'red',
      title: t(($) => $.alerts[titleKey]),
      message: getApiErrorMessage(
        error,
        t(($) => $.alerts[titleKey]),
      ),
    })
  }

  const updateMutation = useMutation({
    ...siteBroadcastsUpdateMutation(),
    onSuccess: async () => {
      await invalidate()
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.broadcastUpdated),
      })
    },
    onError: (error) => notifyError(error, 'broadcastSaveErrorTitle'),
  })

  const sendMutation = useMutation({
    ...siteBroadcastsSendMutation(),
    onSuccess: async () => {
      await invalidate()
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.broadcastSent),
      })
      await navigate({ to: broadcastsReportRoute.to, params: { slug, broadcastId } })
    },
    onError: (error) => notifyError(error, 'broadcastSendErrorTitle'),
  })

  const scheduleMutation = useMutation({
    ...siteBroadcastsScheduleMutation(),
    onSuccess: async () => {
      await invalidate()
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.broadcastScheduled),
      })
      await navigate({ to: broadcastsReportRoute.to, params: { slug, broadcastId } })
    },
    onError: (error) => notifyError(error, 'broadcastSendErrorTitle'),
  })

  if (getQuery.isLoading) return <Loader />

  if (getQuery.isError) {
    return (
      <ApiErrorAlert
        error={getQuery.error}
        title={t(($) => $.alerts.broadcastLoadErrorTitle)}
        fallback={t(($) => $.alerts.broadcastLoadErrorTitle)}
      />
    )
  }

  const isDraft = getQuery.data?.status === 'draft'

  return (
    <Stack>
      <Title order={4}>{t(($) => $.broadcasts.editTitle)}</Title>
      <BroadcastForm
        form={form}
        isPending={updateMutation.isPending}
        onSubmit={(values) =>
          updateMutation.mutate({
            path: { workspaceSlug: slug, id: broadcastId },
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

      <Divider label={t(($) => $.broadcasts.deliveryLabel)} />
      <Text size="sm" c="dimmed">
        {t(($) => $.broadcasts.deliveryHint)}
      </Text>
      <Group align="flex-end">
        <Button
          color="teal"
          disabled={!isDraft}
          loading={sendMutation.isPending}
          onClick={() => sendMutation.mutate({ path: { workspaceSlug: slug, id: broadcastId } })}
        >
          {t(($) => $.broadcasts.sendNow)}
        </Button>
        <TextInput
          type="datetime-local"
          label={t(($) => $.broadcasts.scheduleLabel)}
          value={scheduledAt}
          onChange={(e) => setScheduledAt(e.currentTarget.value)}
        />
        <Button
          variant="light"
          disabled={!isDraft || scheduledAt === ''}
          loading={scheduleMutation.isPending}
          onClick={() =>
            scheduleMutation.mutate({
              path: { workspaceSlug: slug, id: broadcastId },
              body: { scheduledAt: new Date(scheduledAt).toISOString() },
            })
          }
        >
          {t(($) => $.broadcasts.schedule)}
        </Button>
      </Group>
    </Stack>
  )
}
