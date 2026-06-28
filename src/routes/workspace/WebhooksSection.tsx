import {
  Alert,
  Badge,
  Button,
  Card,
  CopyButton,
  Group,
  Loader,
  MultiSelect,
  TextInput,
  Title,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { DataTable } from 'mantine-datatable'
import { useTranslation } from 'react-i18next'
import {
  siteWebhooksCreateMutation,
  siteWebhooksDeleteMutation,
  siteWebhooksListOptions,
  siteWebhooksListQueryKey,
  siteWebhooksUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteWebhookEndpointResource } from '../../generated/site/types.gen.ts'
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'

// The domain events an endpoint can subscribe to. An empty selection means all.
const EVENT_OPTIONS = ['contact.created', 'email.opened', 'email.clicked', 'email.unsubscribed']

export function WebhooksSection({ slug }: { slug: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const confirmDelete = useDeleteConfirmation()

  const queryKey = siteWebhooksListQueryKey({ path: { workspaceSlug: slug } })
  const webhooksQuery = useQuery(siteWebhooksListOptions({ path: { workspaceSlug: slug } }))

  const form = useForm<{ url: string; eventTypes: string[] }>({
    initialValues: { url: '', eventTypes: [] },
  })

  const invalidate = () => queryClient.invalidateQueries({ queryKey })

  const notifyError = (error: unknown, title: string) =>
    notifications.show({
      color: 'red',
      title,
      message: getApiErrorMessage(error as Parameters<typeof getApiErrorMessage>[0], title),
    })

  const createMutation = useMutation({
    ...siteWebhooksCreateMutation(),
    onSuccess: async () => {
      form.reset()
      await invalidate()
    },
    onError: (error) =>
      notifyError(
        error,
        t(($) => $.settings.webhooks.createError),
      ),
  })

  const updateMutation = useMutation({
    ...siteWebhooksUpdateMutation(),
    onSuccess: invalidate,
    onError: (error) =>
      notifyError(
        error,
        t(($) => $.settings.webhooks.saveError),
      ),
  })

  const deleteMutation = useMutation({
    ...siteWebhooksDeleteMutation(),
    onSuccess: invalidate,
    onError: (error) =>
      notifyError(
        error,
        t(($) => $.settings.webhooks.deleteError),
      ),
  })

  const onToggle = (record: SiteWebhookEndpointResource) =>
    updateMutation.mutate({
      path: { workspaceSlug: slug, id: record.id },
      body: { url: record.url, eventTypes: record.eventTypes, enabled: !record.enabled },
    })

  const onDelete = (id: string) =>
    confirmDelete({
      title: t(($) => $.settings.webhooks.deleteConfirmTitle),
      onConfirm: () => deleteMutation.mutate({ path: { workspaceSlug: slug, id } }),
    })

  const records = webhooksQuery.data?.items ?? []

  return (
    <Card withBorder>
      <Title order={4} mb="sm">
        {t(($) => $.settings.webhooksTitle)}
      </Title>

      {webhooksQuery.isError ? (
        <Alert color="red" title={t(($) => $.settings.webhooks.loadError)} mb="md">
          {t(($) => $.settings.webhooks.loadError)}
        </Alert>
      ) : null}

      <form
        onSubmit={form.onSubmit((values) =>
          createMutation.mutate({
            path: { workspaceSlug: slug },
            body: { url: values.url.trim(), eventTypes: values.eventTypes },
          }),
        )}
      >
        <Group align="flex-end" gap="sm" mb="md">
          <TextInput
            label={t(($) => $.settings.webhooks.urlLabel)}
            placeholder="https://example.com/hook"
            required
            w={320}
            {...form.getInputProps('url')}
          />
          <MultiSelect
            label={t(($) => $.settings.webhooks.eventsLabel)}
            description={t(($) => $.settings.webhooks.eventsHint)}
            data={EVENT_OPTIONS}
            w={320}
            {...form.getInputProps('eventTypes')}
          />
          <Button type="submit" loading={createMutation.isPending}>
            {t(($) => $.settings.webhooks.create)}
          </Button>
        </Group>
      </form>

      {webhooksQuery.isLoading ? <Loader /> : null}

      <DataTable
        withTableBorder
        records={records}
        idAccessor="id"
        columns={[
          { accessor: 'url', title: t(($) => $.settings.webhooks.urlColumn) },
          {
            accessor: 'eventTypes',
            title: t(($) => $.settings.webhooks.eventsLabel),
            render: (record) =>
              record.eventTypes.length === 0
                ? t(($) => $.settings.webhooks.allEvents)
                : record.eventTypes.join(', '),
          },
          {
            accessor: 'enabled',
            title: t(($) => $.settings.webhooks.statusColumn),
            render: (record) => (
              <Badge variant="light" color={record.enabled ? 'teal' : 'gray'}>
                {record.enabled
                  ? t(($) => $.settings.webhooks.enabled)
                  : t(($) => $.settings.webhooks.disabled)}
              </Badge>
            ),
          },
          {
            accessor: 'secret',
            title: t(($) => $.settings.webhooks.secret),
            render: (record) => (
              <CopyButton value={record.secret}>
                {({ copied, copy }) => (
                  <Button size="compact-sm" variant="light" onClick={copy}>
                    {copied
                      ? t(($) => $.settings.copied)
                      : t(($) => $.settings.webhooks.copySecret)}
                  </Button>
                )}
              </CopyButton>
            ),
          },
          {
            accessor: 'actions',
            title: '',
            render: (record) => (
              <Group gap="xs" wrap="nowrap">
                <Button size="compact-sm" variant="default" onClick={() => onToggle(record)}>
                  {record.enabled
                    ? t(($) => $.settings.webhooks.disable)
                    : t(($) => $.settings.webhooks.enable)}
                </Button>
                <Button
                  size="compact-sm"
                  color="red"
                  variant="light"
                  onClick={() => onDelete(record.id)}
                >
                  {t(($) => $.settings.webhooks.delete)}
                </Button>
              </Group>
            ),
          },
        ]}
        noRecordsText={t(($) => $.settings.webhooks.empty)}
      />
    </Card>
  )
}
