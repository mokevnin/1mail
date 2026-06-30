import { Badge, Button, Group, Loader, Stack, Text } from '@mantine/core'
import { useCounter } from '@mantine/hooks'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import type { TFunction } from 'i18next'
import { DataTable } from 'mantine-datatable'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteBroadcastsDeleteMutation,
  siteBroadcastsListOptions,
  siteBroadcastsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteBroadcastStatus } from '../../generated/site/types.gen.ts'
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation.tsx'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import {
  broadcastsCreateRoute,
  broadcastsEditRoute,
  broadcastsReportRoute,
  broadcastsRoute,
} from '../../router.tsx'

const PAGE_SIZE = 10

const STATUS_COLORS: Record<SiteBroadcastStatus, string> = {
  draft: 'gray',
  scheduled: 'blue',
  sending: 'yellow',
  sent: 'teal',
  failed: 'red',
}

function StatusBadge({ t, status }: { t: TFunction; status: SiteBroadcastStatus }) {
  return (
    <Badge color={STATUS_COLORS[status]} variant="light">
      {t(($) => $.broadcasts.status[status])}
    </Badge>
  )
}

export function BroadcastsListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const confirmDelete = useDeleteConfirmation()
  const { slug } = broadcastsRoute.useParams()
  const [page, pageHandlers] = useCounter(1, { min: 1 })

  const broadcastsList = useQuery(
    siteBroadcastsListOptions({
      path: { workspaceSlug: slug },
      query: { page, pageSize: PAGE_SIZE },
    }),
  )

  const deleteBroadcastMutation = useResourceMutation({
    mutation: siteBroadcastsDeleteMutation(),
    invalidate: [siteBroadcastsListQueryKey({ path: { workspaceSlug: slug } })],
    successMessage: t(($) => $.notifications.broadcastDeleted),
    errorTitle: t(($) => $.alerts.broadcastDeleteErrorTitle),
  })

  const totalItems = broadcastsList.data?.totalItems ?? 0
  const records = broadcastsList.data?.items ?? []

  const onDeleteClick = (id: string) => {
    deleteBroadcastMutation.reset()
    confirmDelete({
      onConfirm: () => {
        deleteBroadcastMutation.mutate({ path: { workspaceSlug: slug, id } })
      },
    })
  }

  return (
    <Stack>
      <Group justify="flex-end">
        <Button onClick={() => navigate({ to: broadcastsCreateRoute.to, params: { slug } })}>
          {t(($) => $.broadcasts.addBroadcast)}
        </Button>
      </Group>

      {broadcastsList.isError ? (
        <ApiErrorAlert
          error={broadcastsList.error}
          title={t(($) => $.alerts.broadcastLoadErrorTitle)}
          fallback={t(($) => $.alerts.broadcastLoadErrorTitle)}
        />
      ) : null}

      {broadcastsList.isLoading ? <Loader /> : null}

      <DataTable
        withTableBorder
        records={records}
        columns={[
          { accessor: 'name', title: t(($) => $.broadcasts.nameLabel) },
          {
            accessor: 'status',
            title: t(($) => $.broadcasts.statusLabel),
            render: (record) => <StatusBadge t={t} status={record.status} />,
          },
          {
            accessor: 'recipients',
            title: t(($) => $.broadcasts.recipientsLabel),
            render: (record) => record.stats.recipientsTotal,
          },
          {
            accessor: 'opened',
            title: t(($) => $.broadcasts.openedLabel),
            render: (record) => record.stats.openedCount,
          },
          {
            accessor: 'actions',
            title: t(($) => $.table.actions),
            render: (record) => (
              <Group gap="xs">
                <Button
                  size="compact-sm"
                  variant="light"
                  onClick={() =>
                    navigate({
                      to: broadcastsReportRoute.to,
                      params: { slug, broadcastId: record.id },
                    })
                  }
                >
                  {t(($) => $.broadcasts.report)}
                </Button>
                <Button
                  size="compact-sm"
                  disabled={record.status !== 'draft'}
                  onClick={() =>
                    navigate({
                      to: broadcastsEditRoute.to,
                      params: { slug, broadcastId: record.id },
                    })
                  }
                >
                  {t(($) => $.actions.edit)}
                </Button>
                <Button
                  size="compact-sm"
                  color="red"
                  variant="light"
                  onClick={() => onDeleteClick(record.id)}
                >
                  {t(($) => $.actions.delete)}
                </Button>
              </Group>
            ),
          },
        ]}
        totalRecords={totalItems}
        recordsPerPage={PAGE_SIZE}
        page={page}
        onPageChange={pageHandlers.set}
        fetching={broadcastsList.isFetching}
        noRecordsText={t(($) => $.broadcasts.noRecords)}
      />

      {!broadcastsList.isLoading && totalItems === 0 ? (
        <Text>{t(($) => $.broadcasts.emptyState)}</Text>
      ) : null}
    </Stack>
  )
}
