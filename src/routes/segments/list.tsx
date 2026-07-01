import { Button, Group, Loader, Stack, Text } from '@mantine/core'
import { useCounter } from '@mantine/hooks'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import type { TFunction } from 'i18next'
import { DataTable } from 'mantine-datatable'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteSegmentsDeleteMutation,
  siteSegmentsListOptions,
  siteSegmentsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteSegmentType } from '../../generated/site/types.gen.ts'
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation.tsx'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import { segmentsCreateRoute, segmentsEditRoute, segmentsRoute } from '../../router.tsx'

const PAGE_SIZE = 10

function translateType(t: TFunction, type: SiteSegmentType): string {
  return t(($) => $.segments.type[type])
}

export function SegmentsListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const confirmDelete = useDeleteConfirmation()
  const { slug } = segmentsRoute.useParams()
  const [page, pageHandlers] = useCounter(1, { min: 1 })

  const segmentsList = useQuery(
    siteSegmentsListOptions({
      path: { slug: slug },
      query: { page, pageSize: PAGE_SIZE },
    }),
  )

  const deleteSegmentMutation = useResourceMutation({
    mutation: siteSegmentsDeleteMutation(),
    invalidate: [siteSegmentsListQueryKey({ path: { slug: slug } })],
    successMessage: t(($) => $.notifications.segmentDeleted),
    errorTitle: t(($) => $.alerts.segmentDeleteErrorTitle),
  })

  const totalItems = segmentsList.data?.totalItems ?? 0
  const records = segmentsList.data?.items ?? []

  const onDeleteClick = (segmentId: string) => {
    deleteSegmentMutation.reset()
    confirmDelete({
      onConfirm: () => {
        deleteSegmentMutation.mutate({ path: { slug: slug, id: segmentId } })
      },
    })
  }

  return (
    <Stack>
      <Group justify="flex-end">
        <Button onClick={() => navigate({ to: segmentsCreateRoute.to, params: { slug } })}>
          {t(($) => $.segments.addSegment)}
        </Button>
      </Group>

      {segmentsList.isError ? (
        <ApiErrorAlert
          error={segmentsList.error}
          title={t(($) => $.alerts.segmentLoadErrorTitle)}
          fallback={t(($) => $.alerts.segmentLoadErrorTitle)}
        />
      ) : null}

      {segmentsList.isLoading ? <Loader /> : null}

      <DataTable
        withTableBorder
        records={records}
        columns={[
          { accessor: 'name', title: t(($) => $.segments.nameLabel) },
          {
            accessor: 'type',
            title: t(($) => $.segments.typeLabel),
            render: (record) => translateType(t, record.type),
          },
          {
            accessor: 'actions',
            title: t(($) => $.table.actions),
            render: (record) => (
              <Group gap="xs">
                <Button
                  size="compact-sm"
                  onClick={() =>
                    navigate({
                      to: segmentsEditRoute.to,
                      params: { slug, segmentId: record.id },
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
        fetching={segmentsList.isFetching}
        noRecordsText={t(($) => $.segments.noRecords)}
      />

      {!segmentsList.isLoading && totalItems === 0 ? (
        <Text>{t(($) => $.segments.emptyState)}</Text>
      ) : null}
    </Stack>
  )
}
