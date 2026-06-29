import { Badge, Card, Code, Group, Loader, SimpleGrid, Stack, Text, Title } from '@mantine/core'
import { useQuery } from '@tanstack/react-query'
import { DataTable } from 'mantine-datatable'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import { ButtonLink } from '../../components/RouterLink.tsx'
import {
  siteContactsGetOptions,
  siteEventsListOptions,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteContactStatus } from '../../generated/site/types.gen.ts'
import { contactsDetailRoute, contactsEditRoute, contactsRoute } from '../../router.tsx'

const EVENTS_PAGE_SIZE = 10

const STATUS_COLORS: Record<SiteContactStatus, string> = {
  active: 'teal',
  unsubscribed: 'gray',
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <Card withBorder padding="md">
      <Text size="xs" c="dimmed" tt="uppercase">
        {label}
      </Text>
      <Text fw={500}>{value || '—'}</Text>
    </Card>
  )
}

export function ContactDetailPage() {
  const { t } = useTranslation()
  const { slug, contactId } = contactsDetailRoute.useParams()
  const [page, setPage] = useState(1)

  // The component instance is reused across contactId changes, so reset the
  // events page to avoid requesting an out-of-range page for the next contact.
  useEffect(() => {
    setPage(1)
  }, [contactId])

  const contactQuery = useQuery(
    siteContactsGetOptions({ path: { workspaceSlug: slug, id: contactId } }),
  )

  const contact = contactQuery.data

  const eventsQuery = useQuery({
    ...siteEventsListOptions({
      path: { workspaceSlug: slug },
      query: { page, pageSize: EVENTS_PAGE_SIZE, email: contact?.email ?? '' },
    }),
    enabled: Boolean(contact?.email),
  })

  if (contactQuery.isLoading) return <Loader />

  if (contactQuery.isError || !contact) {
    return (
      <ApiErrorAlert
        error={contactQuery.error}
        title={t(($) => $.alerts.contactLoadErrorTitle)}
        fallback={t(($) => $.alerts.contactLoadErrorTitle)}
      />
    )
  }

  const fullName = [contact.firstName, contact.lastName].filter(Boolean).join(' ')
  const customFields = Object.entries(contact.customFields ?? {})
  const eventRecords = eventsQuery.data?.items ?? []
  const eventsTotal = eventsQuery.data?.totalItems ?? 0

  return (
    <Stack>
      <Group justify="space-between" align="flex-start" wrap="wrap">
        <Stack gap={4}>
          <Title order={4}>{contact.email}</Title>
          {fullName ? <Text c="dimmed">{fullName}</Text> : null}
        </Stack>
        <Group gap="xs">
          <Badge color={STATUS_COLORS[contact.status]} variant="light" size="lg">
            {t(($) => $.status[contact.status])}
          </Badge>
          <ButtonLink variant="default" to={contactsEditRoute.to} params={{ slug, contactId }}>
            {t(($) => $.actions.edit)}
          </ButtonLink>
          <ButtonLink variant="subtle" to={contactsRoute.to} params={{ slug }}>
            {t(($) => $.actions.back)}
          </ButtonLink>
        </Group>
      </Group>

      <Title order={5}>{t(($) => $.contacts.detailsTitle)}</Title>
      <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
        <Field label={t(($) => $.table.firstName)} value={contact.firstName ?? ''} />
        <Field label={t(($) => $.table.lastName)} value={contact.lastName ?? ''} />
        <Field label={t(($) => $.table.timeZone)} value={contact.timeZone ?? ''} />
        <Field
          label={t(($) => $.activity.createdAt)}
          value={new Date(contact.createdAt).toLocaleString()}
        />
        <Field
          label={t(($) => $.contacts.updatedAt)}
          value={new Date(contact.updatedAt).toLocaleString()}
        />
      </SimpleGrid>

      {customFields.length > 0 ? (
        <>
          <Title order={5}>{t(($) => $.contacts.customFields)}</Title>
          <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
            {customFields.map(([key, value]) => (
              <Field key={key} label={key} value={value} />
            ))}
          </SimpleGrid>
        </>
      ) : null}

      <Title order={5}>{t(($) => $.contacts.eventsTitle)}</Title>

      {eventsQuery.isError ? (
        <ApiErrorAlert
          error={eventsQuery.error}
          title={t(($) => $.activity.loadErrorTitle)}
          fallback={t(($) => $.activity.loadErrorTitle)}
        />
      ) : null}

      <DataTable
        withTableBorder
        records={eventRecords}
        idAccessor="id"
        columns={[
          {
            accessor: 'createdAt',
            title: t(($) => $.activity.time),
            render: (record) => new Date(record.createdAt).toLocaleString(),
          },
          { accessor: 'action', title: t(($) => $.activity.action) },
        ]}
        rowExpansion={{
          content: ({ record }) => (
            <Stack p="md" gap="xs">
              <Group gap="xs">
                <Text fw={600}>{t(($) => $.activity.subjectId)}:</Text>
                <Text>{record.subjectId}</Text>
              </Group>
              <Group gap="xs">
                <Text fw={600}>{t(($) => $.activity.occurredAt)}:</Text>
                <Text>
                  {record.occurredAt ? new Date(record.occurredAt).toLocaleString() : '—'}
                </Text>
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
        totalRecords={eventsTotal}
        recordsPerPage={EVENTS_PAGE_SIZE}
        page={page}
        onPageChange={setPage}
        fetching={eventsQuery.isFetching}
        noRecordsText={t(($) => $.activity.noRecords)}
      />
    </Stack>
  )
}
