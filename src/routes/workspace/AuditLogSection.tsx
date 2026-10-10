import { Alert, Button, Card, Code, Loader, Stack, Table, Text, Title } from '@mantine/core'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { siteAuditListOptions } from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteAuditEntryResource } from '../../generated/site/types.gen.ts'
import { formatDateTime } from '../../utils/datetime.ts'

const PAGE_SIZE = 25
// The API answers 402 without an Enterprise license and 403 to a plain member; in both
// cases the section is simply not offered.
const HIDDEN_STATUSES = new Set([402, 403])

function isHidden(error: unknown) {
  return (
    typeof error === 'object' &&
    error !== null &&
    'status' in error &&
    typeof error.status === 'number' &&
    HIDDEN_STATUSES.has(error.status)
  )
}

function ActorLabel({ entry }: { entry: SiteAuditEntryResource }) {
  const { t } = useTranslation()
  const { kind, name, id } = entry.actor
  if (kind === 'system') return <>{t(($) => $.settings.auditLog.actorSystem)}</>
  if (kind === 'api_token') {
    return <>{name ?? t(($) => $.settings.auditLog.actorToken, { id: id ?? '' })}</>
  }
  return <>{name ?? id ?? ''}</>
}

function TargetLabel({ entry }: { entry: SiteAuditEntryResource }) {
  const { type, name, id } = entry.target
  return (
    <>
      {type}
      {(name ?? id) ? ` ${name ?? `#${id}`}` : ''}
    </>
  )
}

// AuditPage renders one page of the log and, while more exist, a button that mounts the
// next page beneath it, so the list grows without holding every page in one state.
function AuditPage({ slug, cursor, first }: { slug: string; cursor?: string; first: boolean }) {
  const { t } = useTranslation()
  const [showNext, setShowNext] = useState(false)
  const query = useQuery(
    siteAuditListOptions({
      path: { slug },
      query: { limit: PAGE_SIZE, ...(cursor ? { cursor } : {}) },
    }),
  )

  if (query.isLoading) return <Loader size="sm" />
  if (isHidden(query.error)) return null
  if (query.isError) {
    return (
      <Alert color="red" title={t(($) => $.settings.auditLog.loadError)}>
        {t(($) => $.settings.auditLog.loadError)}
      </Alert>
    )
  }

  const items = query.data?.items ?? []
  const next = query.data?.nextCursor ?? undefined
  if (first && items.length === 0) {
    return (
      <Text c="dimmed" size="sm">
        {t(($) => $.settings.auditLog.empty)}
      </Text>
    )
  }

  return (
    <>
      <Table.ScrollContainer minWidth={640}>
        <Table withTableBorder>
          {first ? (
            <Table.Thead>
              <Table.Tr>
                <Table.Th>{t(($) => $.settings.auditLog.time)}</Table.Th>
                <Table.Th>{t(($) => $.settings.auditLog.actor)}</Table.Th>
                <Table.Th>{t(($) => $.settings.auditLog.action)}</Table.Th>
                <Table.Th>{t(($) => $.settings.auditLog.target)}</Table.Th>
                <Table.Th>{t(($) => $.settings.auditLog.changes)}</Table.Th>
              </Table.Tr>
            </Table.Thead>
          ) : null}
          <Table.Tbody>
            {items.map((entry) => (
              <Table.Tr key={entry.id}>
                <Table.Td>{formatDateTime(entry.occurredAt)}</Table.Td>
                <Table.Td>
                  <ActorLabel entry={entry} />
                </Table.Td>
                <Table.Td>{entry.action}</Table.Td>
                <Table.Td>
                  <TargetLabel entry={entry} />
                </Table.Td>
                <Table.Td>{entry.diff ? <Code>{JSON.stringify(entry.diff)}</Code> : null}</Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      </Table.ScrollContainer>
      {next && !showNext ? (
        <Button variant="light" onClick={() => setShowNext(true)}>
          {t(($) => $.settings.auditLog.loadMore)}
        </Button>
      ) : null}
      {next && showNext ? <AuditPage slug={slug} cursor={next} first={false} /> : null}
    </>
  )
}

// AuditLogSection is the Enterprise Audit log of the workspace: who changed what and
// when, newest first. It renders nothing for a plain member or without a license.
export function AuditLogSection({ slug }: { slug: string }) {
  const { t } = useTranslation()
  const probe = useQuery(siteAuditListOptions({ path: { slug }, query: { limit: PAGE_SIZE } }))
  if (isHidden(probe.error)) return null

  return (
    <Card withBorder>
      <Stack>
        <Title order={4}>{t(($) => $.settings.auditLog.title)}</Title>
        <AuditPage slug={slug} first />
      </Stack>
    </Card>
  )
}
