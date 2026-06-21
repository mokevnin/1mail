import { Alert, Loader, Stack, Title } from '@mantine/core'
import { useCounter } from '@mantine/hooks'
import { useQuery } from '@tanstack/react-query'
import { useParams } from '@tanstack/react-router'
import { DataTable } from 'mantine-datatable'
import { useTranslation } from 'react-i18next'
import { siteEventsListOptions } from '../../generated/site/@tanstack/react-query.gen.ts'

const PAGE_SIZE = 25

// ActivityPage is a read-only, paginated feed of the workspace's tracking events,
// most recent first.
export function ActivityPage() {
  const { t } = useTranslation()
  const { slug } = useParams({ strict: false })
  const [page, pageHandlers] = useCounter(1, { min: 1 })

  const eventsQuery = useQuery({
    ...siteEventsListOptions({
      path: { workspaceSlug: slug ?? '' },
      query: { page, pageSize: PAGE_SIZE },
    }),
    enabled: Boolean(slug),
  })

  const records = eventsQuery.data?.items ?? []
  const totalItems = eventsQuery.data?.totalItems ?? 0

  return (
    <Stack>
      <Title order={2}>{t(($) => $.activity.title)}</Title>

      {eventsQuery.isError ? (
        <Alert color="red" title={t(($) => $.activity.loadErrorTitle)}>
          {t(($) => $.activity.loadErrorTitle)}
        </Alert>
      ) : null}

      {eventsQuery.isLoading ? <Loader /> : null}

      <DataTable
        withTableBorder
        records={records}
        idAccessor="id"
        columns={[
          {
            accessor: 'createdAt',
            title: t(($) => $.activity.time),
            render: (record) => new Date(record.createdAt).toLocaleString(),
          },
          { accessor: 'action', title: t(($) => $.activity.action) },
          {
            accessor: 'subject',
            title: t(($) => $.activity.subject),
            render: (record) => record.email ?? record.subjectId,
          },
          {
            accessor: 'properties',
            title: t(($) => $.activity.properties),
            render: (record) => (record.properties ? JSON.stringify(record.properties) : ''),
          },
        ]}
        totalRecords={totalItems}
        recordsPerPage={PAGE_SIZE}
        page={page}
        onPageChange={pageHandlers.set}
        fetching={eventsQuery.isFetching}
        noRecordsText={t(($) => $.activity.noRecords)}
      />
    </Stack>
  )
}
