import { Alert, Button, Group, Loader, Select, Stack, Text } from '@mantine/core'
import { useCounter } from '@mantine/hooks'
import { modals } from '@mantine/modals'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { contactsCreateRoute, contactsEditRoute } from '../../router.tsx'
import type { TFunction } from 'i18next'
import { DataTable } from 'mantine-datatable'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  siteContactsDeleteMutation,
  siteContactsListOptions,
  siteContactsListQueryKey,
} from '../../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteContactStatus } from '../../../generated/site/types.gen.ts'
import { track } from '../../tracking.ts'

type ContactStatusFilter = 'all' | SiteContactStatus

const PAGE_SIZE = 10

function translateStatus(t: TFunction, status: SiteContactStatus): string {
  if (status === 'active') {
    return t(($) => $.status.active)
  }

  return t(($) => $.status.unsubscribed)
}

export function ContactsListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [page, pageHandlers] = useCounter(1, { min: 1 })
  const [status, setStatus] = useState<ContactStatusFilter>('all')

  const statusForQuery = status === 'all' ? undefined : status

  const contactsList = useQuery(
    siteContactsListOptions({
      query: {
        page,
        pageSize: PAGE_SIZE,
        ...(statusForQuery !== undefined ? { status: statusForQuery } : {}),
      },
    }),
  )

  const deleteContactMutation = useMutation({
    ...siteContactsDeleteMutation(),
    onSuccess: async (_result, variables) => {
      await queryClient.invalidateQueries({ queryKey: siteContactsListQueryKey() })
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
        message: error.detail ?? error.title,
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
    modals.openConfirmModal({
      title: t(($) => $.form.deleteTitle),
      children: <Text>{t(($) => $.form.deleteDescription)}</Text>,
      labels: {
        cancel: t(($) => $.actions.cancel),
        confirm: t(($) => $.actions.delete),
      },
      confirmProps: { color: 'red' },
      onConfirm: () => {
        deleteContactMutation.mutate({ path: { id: contactId } })
      },
    })
  }

  return (
    <Stack>
      <Group justify="space-between" align="flex-end">
        <Select
          label={t(($) => $.contacts.statusFilter)}
          data={[
            { value: 'all', label: t(($) => $.status.all) },
            { value: 'active', label: t(($) => $.status.active) },
            { value: 'unsubscribed', label: t(($) => $.status.unsubscribed) },
          ]}
          value={status}
          onChange={onStatusChange}
          w={220}
        />

        <Button onClick={() => navigate({ to: contactsCreateRoute.to })}>
          {t(($) => $.contacts.addContact)}
        </Button>
      </Group>

      {contactsList.isError ? (
        <Alert color="red" title={t(($) => $.alerts.loadErrorTitle)}>
          {contactsList.error.detail ?? contactsList.error.title}
        </Alert>
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
                      params: { contactId: record.id },
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
