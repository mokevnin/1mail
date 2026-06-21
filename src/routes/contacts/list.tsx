import { track } from '@1mail/analytics'
import { Button, Group, Loader, Select, Stack, Text } from '@mantine/core'
import { useCounter } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import type { TFunction } from 'i18next'
import { DataTable } from 'mantine-datatable'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteContactsDeleteMutation,
  siteContactsListOptions,
  siteContactsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { SiteContactStatus } from '../../generated/site/types.gen.ts'
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation.tsx'
import { contactsCreateRoute, contactsEditRoute, contactsRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'

type ContactStatusFilter = 'all' | SiteContactStatus

const PAGE_SIZE = 10
const contactStatusFilterValues: ContactStatusFilter[] = [
  'all',
  ...Object.values(SiteContactStatus),
]

function translateStatus(t: TFunction, status: ContactStatusFilter): string {
  return t(($) => $.status[status])
}

export function ContactsListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const confirmDelete = useDeleteConfirmation()
  const { slug } = contactsRoute.useParams()
  const [page, pageHandlers] = useCounter(1, { min: 1 })
  const [status, setStatus] = useState<ContactStatusFilter>('all')

  const contactsList = useQuery(
    siteContactsListOptions({
      path: { workspaceSlug: slug },
      query: {
        page,
        pageSize: PAGE_SIZE,
        ...(status !== 'all' ? { status } : {}),
      },
    }),
  )

  const deleteContactMutation = useMutation({
    ...siteContactsDeleteMutation(),
    onSuccess: async (_result, variables) => {
      await queryClient.invalidateQueries({
        queryKey: siteContactsListQueryKey({ path: { workspaceSlug: slug } }),
      })
      await track('contact.deleted', { contactId: variables.path.id })
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.contactDeleted),
      })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.alerts.deleteErrorTitle),
        message: getApiErrorMessage(
          error,
          t(($) => $.alerts.deleteErrorTitle),
        ),
      })
    },
  })

  const totalItems = contactsList.data?.totalItems ?? 0
  const records = contactsList.data?.items ?? []

  const onStatusChange = (value: string | null) => {
    setStatus((value as ContactStatusFilter | null) ?? 'all')
    pageHandlers.set(1)
  }

  const onDeleteClick = (contactId: string) => {
    deleteContactMutation.reset()
    confirmDelete({
      onConfirm: () => {
        deleteContactMutation.mutate({ path: { workspaceSlug: slug, id: contactId } })
      },
    })
  }

  return (
    <Stack>
      <Group justify="space-between" align="flex-end" wrap="wrap">
        <Select
          label={t(($) => $.contacts.statusFilter)}
          data={contactStatusFilterValues.map((value) => ({
            value,
            label: translateStatus(t, value),
          }))}
          value={status}
          onChange={onStatusChange}
          w={{ base: '100%', xs: 220 }}
        />

        <Button onClick={() => navigate({ to: contactsCreateRoute.to, params: { slug } })}>
          {t(($) => $.contacts.addContact)}
        </Button>
      </Group>

      {contactsList.isError ? (
        <ApiErrorAlert
          error={contactsList.error}
          title={t(($) => $.alerts.loadErrorTitle)}
          fallback={t(($) => $.alerts.loadErrorTitle)}
        />
      ) : null}

      {contactsList.isLoading ? <Loader /> : null}

      <DataTable
        withTableBorder
        records={records}
        columns={[
          { accessor: 'email', title: t(($) => $.table.email) },
          { accessor: 'firstName', title: t(($) => $.table.firstName) },
          { accessor: 'lastName', title: t(($) => $.table.lastName) },
          { accessor: 'timeZone', title: t(($) => $.table.timeZone) },
          {
            accessor: 'status',
            title: t(($) => $.table.status),
            render: (record) => translateStatus(t, record.status),
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
                      to: contactsEditRoute.to,
                      params: { slug, contactId: record.id },
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
        fetching={contactsList.isFetching}
        noRecordsText={t(($) => $.contacts.noRecords)}
      />

      {!contactsList.isLoading && totalItems === 0 ? (
        <Text>{t(($) => $.contacts.emptyState)}</Text>
      ) : null}
    </Stack>
  )
}
