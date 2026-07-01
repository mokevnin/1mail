import { Card, Code, Group, Loader, SimpleGrid, Stack, Text, Title } from '@mantine/core'
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
import { contactsDetailRoute, contactsEditRoute, contactsRoute } from '../../router.tsx'
import { formatDateTime } from '../../utils/datetime.ts'

const EVENTS_PAGE_SIZE = 10

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

// Custom field values are typed (string/number/bool/…); render scalars directly and
// anything richer as compact JSON.
function formatCustomValue(value: unknown): string {
  if (value == null) return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
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

  const contactQuery = useQuery(siteContactsGetOptions({ path: { slug: slug, id: contactId } }))

  const contact = contactQuery.data

  // Activity is keyed by the stable contact_id (ADR 0002): this also surfaces
  // anonymous events stitched onto the contact at Identify, which an email filter
  // would miss.
  const eventsQuery = useQuery(
    siteEventsListOptions({
      path: { slug: slug },
      query: { page, pageSize: EVENTS_PAGE_SIZE, contactId },
    }),
  )

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
          <Title order={4}>{contact.email ?? contact.subjectId ?? contact.phone ?? '—'}</Title>
          {fullName ? <Text c="dimmed">{fullName}</Text> : null}
        </Stack>
        <Group gap="xs">
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
        <Field label={t(($) => $.table.email)} value={contact.email ?? ''} />
        <Field label={t(($) => $.table.subjectId)} value={contact.subjectId ?? ''} />
        <Field label={t(($) => $.table.phone)} value={contact.phone ?? ''} />
        <Field label={t(($) => $.table.firstName)} value={contact.firstName ?? ''} />
        <Field label={t(($) => $.table.lastName)} value={contact.lastName ?? ''} />
        <Field label={t(($) => $.table.timeZone)} value={contact.timeZone ?? ''} />
        <Field label={t(($) => $.activity.createdAt)} value={formatDateTime(contact.createdAt)} />
        <Field label={t(($) => $.contacts.updatedAt)} value={formatDateTime(contact.updatedAt)} />
      </SimpleGrid>

      {customFields.length > 0 ? (
        <>
          <Title order={5}>{t(($) => $.contacts.customFields)}</Title>
          <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
            {customFields.map(([key, value]) => (
              <Field key={key} label={key} value={formatCustomValue(value)} />
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
            render: (record) => formatDateTime(record.createdAt),
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
                <Text>{record.occurredAt ? formatDateTime(record.occurredAt) : '—'}</Text>
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
