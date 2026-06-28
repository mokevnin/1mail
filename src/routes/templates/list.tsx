import { Button, Group, Loader, Stack, Text } from '@mantine/core'
import { useCounter } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { DataTable } from 'mantine-datatable'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteTemplatesDeleteMutation,
  siteTemplatesListOptions,
  siteTemplatesListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation.tsx'
import { templatesCreateRoute, templatesEditRoute, templatesRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'

const PAGE_SIZE = 10

export function TemplatesListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const confirmDelete = useDeleteConfirmation()
  const { slug } = templatesRoute.useParams()
  const [page, pageHandlers] = useCounter(1, { min: 1 })

  const templatesList = useQuery(
    siteTemplatesListOptions({
      path: { workspaceSlug: slug },
      query: { page, pageSize: PAGE_SIZE },
    }),
  )

  const deleteMutation = useMutation({
    ...siteTemplatesDeleteMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: siteTemplatesListQueryKey({ path: { workspaceSlug: slug } }),
      })
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.templateDeleted),
      })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.alerts.templateDeleteErrorTitle),
        message: getApiErrorMessage(
          error,
          t(($) => $.alerts.templateDeleteErrorTitle),
        ),
      })
    },
  })

  const totalItems = templatesList.data?.totalItems ?? 0
  const records = templatesList.data?.items ?? []

  const onDeleteClick = (id: string) => {
    deleteMutation.reset()
    confirmDelete({ onConfirm: () => deleteMutation.mutate({ path: { workspaceSlug: slug, id } }) })
  }

  return (
    <Stack>
      <Group justify="flex-end">
        <Button onClick={() => navigate({ to: templatesCreateRoute.to, params: { slug } })}>
          {t(($) => $.templates.addTemplate)}
        </Button>
      </Group>

      {templatesList.isError ? (
        <ApiErrorAlert
          error={templatesList.error}
          title={t(($) => $.alerts.templateLoadErrorTitle)}
          fallback={t(($) => $.alerts.templateLoadErrorTitle)}
        />
      ) : null}

      {templatesList.isLoading ? <Loader /> : null}

      <DataTable
        withTableBorder
        records={records}
        columns={[
          { accessor: 'name', title: t(($) => $.templates.nameLabel) },
          { accessor: 'subject', title: t(($) => $.templates.subjectLabel) },
          {
            accessor: 'actions',
            title: t(($) => $.table.actions),
            render: (record) => (
              <Group gap="xs">
                <Button
                  size="compact-sm"
                  onClick={() =>
                    navigate({ to: templatesEditRoute.to, params: { slug, templateId: record.id } })
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
        fetching={templatesList.isFetching}
        noRecordsText={t(($) => $.templates.noRecords)}
      />

      {!templatesList.isLoading && totalItems === 0 ? (
        <Text>{t(($) => $.templates.emptyState)}</Text>
      ) : null}
    </Stack>
  )
}
