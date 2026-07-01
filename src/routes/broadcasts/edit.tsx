import { Button, Divider, Group, Loader, Stack, Text, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { useQuery } from '@tanstack/react-query'
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
  siteBroadcastsTestSendMutation,
  siteBroadcastsUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteBroadcastResource } from '../../generated/site/types.gen.ts'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import { broadcastsEditRoute, broadcastsReportRoute } from '../../router.tsx'
import { BroadcastForm, type BroadcastFormValues } from './BroadcastForm.tsx'

export function BroadcastEditPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug, broadcastId } = broadcastsEditRoute.useParams()
  const [scheduledAt, setScheduledAt] = useState('')
  const [testEmail, setTestEmail] = useState('')

  const form = useForm<BroadcastFormValues>({
    initialValues: {
      name: '',
      subject: '',
      fromName: '',
      fromEmail: '',
      body: '',
      segmentId: '',
    },
  })

  const getQuery = useQuery(siteBroadcastsGetOptions({ path: { slug: slug, id: broadcastId } }))

  const applyData = useEffectEvent((data: SiteBroadcastResource | undefined) => {
    if (!data) return
    form.setValues({
      name: data.name,
      subject: data.subject,
      fromName: data.fromName ?? '',
      fromEmail: data.fromEmail ?? '',
      body: data.body,
      segmentId: data.segmentId ?? '',
    })
  })

  useEffect(() => {
    applyData(getQuery.data)
  }, [getQuery.data])

  const invalidateKeys = [
    siteBroadcastsListQueryKey({ path: { slug: slug } }),
    siteBroadcastsGetQueryKey({ path: { slug: slug, id: broadcastId } }),
  ]

  const toReport = () => navigate({ to: broadcastsReportRoute.to, params: { slug, broadcastId } })

  const updateMutation = useResourceMutation({
    mutation: siteBroadcastsUpdateMutation(),
    invalidate: invalidateKeys,
    successMessage: t(($) => $.notifications.broadcastUpdated),
    errorTitle: t(($) => $.alerts.broadcastSaveErrorTitle),
  })

  const sendMutation = useResourceMutation({
    mutation: siteBroadcastsSendMutation(),
    invalidate: invalidateKeys,
    successMessage: t(($) => $.notifications.broadcastSent),
    errorTitle: t(($) => $.alerts.broadcastSendErrorTitle),
    onDone: toReport,
  })

  const scheduleMutation = useResourceMutation({
    mutation: siteBroadcastsScheduleMutation(),
    invalidate: invalidateKeys,
    successMessage: t(($) => $.notifications.broadcastScheduled),
    errorTitle: t(($) => $.alerts.broadcastSendErrorTitle),
    onDone: toReport,
  })

  const testSendMutation = useResourceMutation({
    mutation: siteBroadcastsTestSendMutation(),
    successMessage: t(($) => $.notifications.testSent),
    errorTitle: t(($) => $.alerts.broadcastSendErrorTitle),
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
            path: { slug: slug, id: broadcastId },
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

      <Divider label={t(($) => $.broadcasts.deliveryLabel)} />
      <Text size="sm" c="dimmed">
        {t(($) => $.broadcasts.deliveryHint)}
      </Text>
      <Group align="flex-end">
        <Button
          color="teal"
          disabled={!isDraft}
          loading={sendMutation.isPending}
          onClick={() => sendMutation.mutate({ path: { slug: slug, id: broadcastId } })}
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
              path: { slug: slug, id: broadcastId },
              body: { scheduledAt: new Date(scheduledAt).toISOString() },
            })
          }
        >
          {t(($) => $.broadcasts.schedule)}
        </Button>
      </Group>

      <Group align="flex-end">
        <TextInput
          type="email"
          label={t(($) => $.broadcasts.testSendEmailLabel)}
          value={testEmail}
          onChange={(e) => setTestEmail(e.currentTarget.value)}
        />
        <Button
          variant="default"
          disabled={testEmail === ''}
          loading={testSendMutation.isPending}
          onClick={() =>
            testSendMutation.mutate({
              path: { slug: slug, id: broadcastId },
              body: { email: testEmail },
            })
          }
        >
          {t(($) => $.broadcasts.testSendButton)}
        </Button>
      </Group>
    </Stack>
  )
}
