import { Badge, Button, Group, Loader, Stack, Text } from '@mantine/core'
import { useCounter } from '@mantine/hooks'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { DataTable } from 'mantine-datatable'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteAutomationsActivateMutation,
  siteAutomationsDeactivateMutation,
  siteAutomationsDeleteMutation,
  siteAutomationsListOptions,
  siteAutomationsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation.tsx'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import { automationsCreateRoute, automationsEditRoute, automationsRoute } from '../../router.tsx'

const PAGE_SIZE = 10

export function AutomationsListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const confirmDelete = useDeleteConfirmation()
  const { slug } = automationsRoute.useParams()
  const [page, pageHandlers] = useCounter(1, { min: 1 })

  const automationsList = useQuery(
    siteAutomationsListOptions({
      path: { workspaceSlug: slug },
      query: { page, pageSize: PAGE_SIZE },
    }),
  )

  const listQueryKey = siteAutomationsListQueryKey({ path: { workspaceSlug: slug } })

  const deleteMutation = useResourceMutation({
    mutation: siteAutomationsDeleteMutation(),
    invalidate: [listQueryKey],
    successMessage: t(($) => $.notifications.automationDeleted),
    errorTitle: t(($) => $.alerts.automationDeleteErrorTitle),
  })

  const activateMutation = useResourceMutation({
    mutation: siteAutomationsActivateMutation(),
    invalidate: [listQueryKey],
    successMessage: t(($) => $.notifications.automationActivated),
    errorTitle: t(($) => $.alerts.automationStatusErrorTitle),
  })

  const deactivateMutation = useResourceMutation({
    mutation: siteAutomationsDeactivateMutation(),
    invalidate: [listQueryKey],
    successMessage: t(($) => $.notifications.automationDeactivated),
    errorTitle: t(($) => $.alerts.automationStatusErrorTitle),
  })

  const totalItems = automationsList.data?.totalItems ?? 0
  const records = automationsList.data?.items ?? []

  const onDeleteClick = (id: string) => {
    deleteMutation.reset()
    confirmDelete({ onConfirm: () => deleteMutation.mutate({ path: { workspaceSlug: slug, id } }) })
  }

  return (
    <Stack>
      <Group justify="flex-end">
        <Button onClick={() => navigate({ to: automationsCreateRoute.to, params: { slug } })}>
          {t(($) => $.automations.addAutomation)}
        </Button>
      </Group>

      {automationsList.isError ? (
        <ApiErrorAlert
          error={automationsList.error}
          title={t(($) => $.alerts.automationLoadErrorTitle)}
          fallback={t(($) => $.alerts.automationLoadErrorTitle)}
        />
      ) : null}

      {automationsList.isLoading ? <Loader /> : null}

      <DataTable
        withTableBorder
        records={records}
        columns={[
          { accessor: 'name', title: t(($) => $.automations.nameLabel) },
          { accessor: 'triggerEvent', title: t(($) => $.automations.triggerColumn) },
          {
            accessor: 'status',
            title: t(($) => $.automations.statusLabel),
            render: (record) => (
              <Badge variant="light" color={record.status === 'active' ? 'teal' : 'gray'}>
                {record.status === 'active'
                  ? t(($) => $.automations.statusActive)
                  : t(($) => $.automations.statusDraft)}
              </Badge>
            ),
          },
          {
            accessor: 'actions',
            title: t(($) => $.table.actions),
            render: (record) => (
              <Group gap="xs">
                {record.status === 'active' ? (
                  <Button
                    size="compact-sm"
                    variant="light"
                    color="orange"
                    loading={deactivateMutation.isPending}
                    onClick={() =>
                      deactivateMutation.mutate({ path: { workspaceSlug: slug, id: record.id } })
                    }
                  >
                    {t(($) => $.automations.deactivate)}
                  </Button>
                ) : (
                  <Button
                    size="compact-sm"
                    variant="light"
                    color="teal"
                    loading={activateMutation.isPending}
                    onClick={() =>
                      activateMutation.mutate({ path: { workspaceSlug: slug, id: record.id } })
                    }
                  >
                    {t(($) => $.automations.activate)}
                  </Button>
                )}
                <Button
                  size="compact-sm"
                  onClick={() =>
                    navigate({
                      to: automationsEditRoute.to,
                      params: { slug, automationId: record.id },
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
        fetching={automationsList.isFetching}
        noRecordsText={t(($) => $.automations.noRecords)}
      />

      {!automationsList.isLoading && totalItems === 0 ? (
        <Text>{t(($) => $.automations.emptyState)}</Text>
      ) : null}
    </Stack>
  )
}
