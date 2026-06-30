import { Badge, Loader, Stack, Text, Title } from '@mantine/core'
import { useCounter } from '@mantine/hooks'
import { useQuery } from '@tanstack/react-query'
import type { TFunction } from 'i18next'
import { DataTable } from 'mantine-datatable'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import { siteTransactionalEmailsListOptions } from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteTransactionalEmailStatus } from '../../generated/site/types.gen.ts'
import { transactionalEmailsRoute } from '../../router.tsx'

const PAGE_SIZE = 20

const STATUS_COLORS: Record<SiteTransactionalEmailStatus, string> = {
  pending: 'yellow',
  sent: 'teal',
  suppressed: 'gray',
  failed: 'red',
}

function StatusBadge({ t, status }: { t: TFunction; status: SiteTransactionalEmailStatus }) {
  return (
    <Badge color={STATUS_COLORS[status]} variant="light">
      {t(($) => $.transactionalEmails.status[status])}
    </Badge>
  )
}

export function TransactionalEmailsListPage() {
  const { t } = useTranslation()
  const { slug } = transactionalEmailsRoute.useParams()
  const [page, pageHandlers] = useCounter(1, { min: 1 })

  const list = useQuery(
    siteTransactionalEmailsListOptions({
      path: { workspaceSlug: slug },
      query: { page, pageSize: PAGE_SIZE },
    }),
  )

  const totalItems = list.data?.totalItems ?? 0
  const records = list.data?.items ?? []

  return (
    <Stack>
      <Title order={2}>{t(($) => $.transactionalEmails.title)}</Title>
      <Text c="dimmed" size="sm">
        {t(($) => $.transactionalEmails.subtitle)}
      </Text>

      {list.isError ? (
        <ApiErrorAlert
          error={list.error}
          title={t(($) => $.alerts.transactionalEmailsLoadErrorTitle)}
          fallback={t(($) => $.alerts.transactionalEmailsLoadErrorTitle)}
        />
      ) : null}

      {list.isLoading ? <Loader /> : null}

      <DataTable
        withTableBorder
        records={records}
        columns={[
          { accessor: 'destination', title: t(($) => $.transactionalEmails.destinationLabel) },
          {
            accessor: 'status',
            title: t(($) => $.transactionalEmails.statusLabel),
            render: (record) => <StatusBadge t={t} status={record.status} />,
          },
          {
            accessor: 'error',
            title: t(($) => $.transactionalEmails.errorLabel),
            render: (record) => record.error ?? '',
          },
          {
            accessor: 'createdAt',
            title: t(($) => $.transactionalEmails.sentAtLabel),
            render: (record) => new Date(record.createdAt).toLocaleString(),
          },
        ]}
        totalRecords={totalItems}
        recordsPerPage={PAGE_SIZE}
        page={page}
        onPageChange={pageHandlers.set}
        fetching={list.isFetching}
        noRecordsText={t(($) => $.transactionalEmails.noRecords)}
      />
    </Stack>
  )
}
