import { Alert, Code, Group, Loader, Stack, Switch, Text, TextInput, Title } from '@mantine/core'
import { useDebouncedValue } from '@mantine/hooks'
import { useQuery } from '@tanstack/react-query'
import { useParams } from '@tanstack/react-router'
import { DataTable } from 'mantine-datatable'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { siteEventsListOptions } from '../../generated/site/@tanstack/react-query.gen.ts'
import { formatDateTime } from '../../utils/datetime.ts'

const PAGE_SIZE = 25
const LIVE_REFETCH_MS = 5000

// ActivityPage is a read-only feed of the workspace's tracking events, most
// recent first. Live mode polls so newly tracked events surface without a
// manual reload — the primary way to verify that tracking works.
export function ActivityPage() {
  const { t } = useTranslation()
  const { slug } = useParams({ strict: false })
  const [page, setPage] = useState(1)
  const [live, setLive] = useState(true)
  const [actionFilter, setActionFilter] = useState('')
  const [debouncedAction] = useDebouncedValue(actionFilter.trim(), 300)

  // Reset to the first page whenever the action filter changes.
  useEffect(() => {
    setPage(1)
  }, [debouncedAction])

  const eventsQuery = useQuery({
    ...siteEventsListOptions({
      path: { slug: slug ?? '' },
      query: {
        page,
        pageSize: PAGE_SIZE,
        ...(debouncedAction ? { action: debouncedAction } : {}),
      },
    }),
    enabled: Boolean(slug),
    refetchInterval: live ? LIVE_REFETCH_MS : false,
  })

  const records = eventsQuery.data?.items ?? []
  const totalItems = eventsQuery.data?.totalItems ?? 0

  return (
    <Stack>
      <Group justify="space-between" align="center">
        <Title order={2}>{t(($) => $.activity.title)}</Title>
        <Group gap="md">
          <TextInput
            placeholder={t(($) => $.activity.filterActionPlaceholder)}
            value={actionFilter}
            onChange={(event) => setActionFilter(event.currentTarget.value)}
          />
          <Switch
            label={t(($) => $.activity.live)}
            checked={live}
            onChange={(event) => setLive(event.currentTarget.checked)}
          />
        </Group>
      </Group>

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
            render: (record) => formatDateTime(record.createdAt),
          },
          { accessor: 'action', title: t(($) => $.activity.action) },
          {
            accessor: 'subject',
            title: t(($) => $.activity.subject),
            render: (record) => record.email ?? record.subjectId,
          },
        ]}
        rowExpansion={{
          content: ({ record }) => (
            <Stack p="md" gap="xs">
              <Text fw={700}>{t(($) => $.activity.details)}</Text>
              <Group gap="xs">
                <Text fw={600}>{t(($) => $.activity.subjectId)}:</Text>
                <Text>{record.subjectId}</Text>
              </Group>
              <Group gap="xs">
                <Text fw={600}>{t(($) => $.activity.occurredAt)}:</Text>
                <Text>{record.occurredAt ? formatDateTime(record.occurredAt) : '—'}</Text>
              </Group>
              <Group gap="xs">
                <Text fw={600}>{t(($) => $.activity.createdAt)}:</Text>
                <Text>{formatDateTime(record.createdAt)}</Text>
              </Group>
              <Stack gap="xs">
                <Text fw={600}>{t(($) => $.activity.properties)}:</Text>
                <Code block>
                  {record.properties ? JSON.stringify(record.properties, null, 2) : '{}'}
                </Code>
              </Stack>
            </Stack>
          ),
        }}
        totalRecords={totalItems}
        recordsPerPage={PAGE_SIZE}
        page={page}
        onPageChange={setPage}
        fetching={eventsQuery.isFetching}
        noRecordsText={t(($) => $.activity.noRecords)}
      />
    </Stack>
  )
}
