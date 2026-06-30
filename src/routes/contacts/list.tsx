import { ActionIcon, Button, Group, Loader, Stack, Text, Tooltip } from '@mantine/core'
import { useCounter } from '@mantine/hooks'
import { IconEye, IconPencil, IconTrash } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { DataTable } from 'mantine-datatable'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import { ActionIconLink } from '../../components/RouterLink.tsx'
import {
  siteContactsDeleteMutation,
  siteContactsListOptions,
  siteContactsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation.tsx'
import { useResourceMutation } from '../../hooks/useResourceMutation.ts'
import {
  contactsCreateRoute,
  contactsDetailRoute,
  contactsEditRoute,
  contactsRoute,
} from '../../router.tsx'

const PAGE_SIZE = 10

export function ContactsListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const confirmDelete = useDeleteConfirmation()
  const { slug } = contactsRoute.useParams()
  const [page, pageHandlers] = useCounter(1, { min: 1 })

  const contactsList = useQuery(
    siteContactsListOptions({
      path: { workspaceSlug: slug },
      query: {
        page,
        pageSize: PAGE_SIZE,
      },
    }),
  )

  const deleteContactMutation = useResourceMutation({
    mutation: siteContactsDeleteMutation(),
    invalidate: [siteContactsListQueryKey({ path: { workspaceSlug: slug } })],
    successMessage: t(($) => $.notifications.contactDeleted),
    errorTitle: t(($) => $.alerts.deleteErrorTitle),
  })

  const totalItems = contactsList.data?.totalItems ?? 0
  const records = contactsList.data?.items ?? []

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
      <Group justify="flex-end" align="flex-end" wrap="wrap">
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
            accessor: 'actions',
            title: t(($) => $.table.actions),
            render: (record) => (
              <Group gap="xs" wrap="nowrap">
                <Tooltip label={t(($) => $.actions.view)}>
                  <ActionIconLink
                    variant="light"
                    to={contactsDetailRoute.to}
                    params={{ slug, contactId: record.id }}
                    aria-label={t(($) => $.actions.view)}
                  >
                    <IconEye size={16} />
                  </ActionIconLink>
                </Tooltip>
                <Tooltip label={t(($) => $.actions.edit)}>
                  <ActionIconLink
                    variant="light"
                    to={contactsEditRoute.to}
                    params={{ slug, contactId: record.id }}
                    aria-label={t(($) => $.actions.edit)}
                  >
                    <IconPencil size={16} />
                  </ActionIconLink>
                </Tooltip>
                <Tooltip label={t(($) => $.actions.delete)}>
                  <ActionIcon
                    variant="light"
                    color="red"
                    onClick={() => onDeleteClick(record.id)}
                    aria-label={t(($) => $.actions.delete)}
                  >
                    <IconTrash size={16} />
                  </ActionIcon>
                </Tooltip>
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
