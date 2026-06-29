import { Alert, Badge, Button, Card, Group, Loader, TextInput, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { useCounter } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { DataTable } from 'mantine-datatable'
import { useTranslation } from 'react-i18next'
import {
  siteSuppressionsCreateMutation,
  siteSuppressionsDeleteMutation,
  siteSuppressionsListOptions,
  siteSuppressionsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteSuppressionReason } from '../../generated/site/types.gen.ts'
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'

const REASON_COLORS: Record<SiteSuppressionReason, string> = {
  unsubscribed: 'yellow',
  bounce: 'orange',
  complaint: 'red',
  manual: 'gray',
}

// The list grows automatically (bounces/complaints/unsubscribes), so it is
// paginated server-side rather than rendering only the first page.
const PAGE_SIZE = 20

// SuppressionsSection manages the workspace do-not-send list: unsubscribes,
// bounces, and complaints land here automatically; addresses can also be added
// manually. The send path skips every address listed here.
export function SuppressionsSection({ slug }: { slug: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const confirmDelete = useDeleteConfirmation()
  const [page, pageHandlers] = useCounter(1, { min: 1 })

  const queryKey = siteSuppressionsListQueryKey({ path: { workspaceSlug: slug } })
  const suppressionsQuery = useQuery(
    siteSuppressionsListOptions({
      path: { workspaceSlug: slug },
      query: { page, pageSize: PAGE_SIZE },
    }),
  )

  const form = useForm<{ email: string }>({ initialValues: { email: '' } })

  const invalidate = () => queryClient.invalidateQueries({ queryKey })

  const notifyError = (error: unknown, title: string) =>
    notifications.show({
      color: 'red',
      title,
      message: getApiErrorMessage(error as Parameters<typeof getApiErrorMessage>[0], title),
    })

  const createMutation = useMutation({
    ...siteSuppressionsCreateMutation(),
    onSuccess: async () => {
      form.reset()
      pageHandlers.set(1) // newest is first (id desc), so jump to page 1 to show it
      await invalidate()
    },
    onError: (error) =>
      notifyError(
        error,
        t(($) => $.settings.suppressions.createError),
      ),
  })

  const deleteMutation = useMutation({
    ...siteSuppressionsDeleteMutation(),
    onSuccess: invalidate,
    onError: (error) =>
      notifyError(
        error,
        t(($) => $.settings.suppressions.deleteError),
      ),
  })

  const onDelete = (id: string) =>
    confirmDelete({
      title: t(($) => $.settings.suppressions.deleteConfirmTitle),
      onConfirm: () => deleteMutation.mutate({ path: { workspaceSlug: slug, id } }),
    })

  const records = suppressionsQuery.data?.items ?? []
  const totalItems = suppressionsQuery.data?.totalItems ?? 0

  return (
    <Card withBorder>
      <Title order={4} mb="sm">
        {t(($) => $.settings.suppressions.title)}
      </Title>

      {suppressionsQuery.isError ? (
        <Alert color="red" title={t(($) => $.settings.suppressions.loadError)} mb="md">
          {t(($) => $.settings.suppressions.loadError)}
        </Alert>
      ) : null}

      <form
        onSubmit={form.onSubmit((values) =>
          createMutation.mutate({
            path: { workspaceSlug: slug },
            body: { email: values.email.trim() },
          }),
        )}
      >
        <Group align="flex-end" gap="sm" mb="md">
          <TextInput
            label={t(($) => $.settings.suppressions.emailLabel)}
            description={t(($) => $.settings.suppressions.emailHint)}
            placeholder="blocked@example.com"
            type="email"
            required
            w={320}
            {...form.getInputProps('email')}
          />
          <Button type="submit" loading={createMutation.isPending}>
            {t(($) => $.settings.suppressions.create)}
          </Button>
        </Group>
      </form>

      {suppressionsQuery.isLoading ? <Loader /> : null}

      <DataTable
        withTableBorder
        records={records}
        idAccessor="id"
        columns={[
          { accessor: 'email', title: t(($) => $.settings.suppressions.emailColumn) },
          {
            accessor: 'reason',
            title: t(($) => $.settings.suppressions.reasonColumn),
            render: (record) => (
              <Badge variant="light" color={REASON_COLORS[record.reason]}>
                {t(($) => $.settings.suppressions.reasons[record.reason])}
              </Badge>
            ),
          },
          {
            accessor: 'actions',
            title: '',
            render: (record) => (
              <Group justify="flex-end">
                <Button
                  size="compact-sm"
                  color="red"
                  variant="light"
                  onClick={() => onDelete(record.id)}
                >
                  {t(($) => $.settings.suppressions.delete)}
                </Button>
              </Group>
            ),
          },
        ]}
        noRecordsText={t(($) => $.settings.suppressions.empty)}
        totalRecords={totalItems}
        recordsPerPage={PAGE_SIZE}
        page={page}
        onPageChange={pageHandlers.set}
      />
    </Card>
  )
}
