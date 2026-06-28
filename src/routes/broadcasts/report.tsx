import { Badge, Card, Group, Loader, SimpleGrid, Stack, Text, Title } from '@mantine/core'
import { useQuery } from '@tanstack/react-query'
import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import { siteBroadcastsGetOptions } from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteBroadcastStats, SiteBroadcastStatus } from '../../generated/site/types.gen.ts'
import { broadcastsReportRoute } from '../../router.tsx'

const STATUS_COLORS: Record<SiteBroadcastStatus, string> = {
  draft: 'gray',
  scheduled: 'blue',
  sending: 'yellow',
  sent: 'teal',
  failed: 'red',
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <Card withBorder padding="md">
      <Text size="xs" c="dimmed" tt="uppercase">
        {label}
      </Text>
      <Text fw={700} size="xl">
        {value}
      </Text>
    </Card>
  )
}

function statEntries(t: TFunction, stats: SiteBroadcastStats): { label: string; value: number }[] {
  return [
    { label: t(($) => $.broadcasts.stats.recipients), value: stats.recipientsTotal },
    { label: t(($) => $.broadcasts.stats.sent), value: stats.sentCount },
    { label: t(($) => $.broadcasts.stats.opened), value: stats.openedCount },
    { label: t(($) => $.broadcasts.stats.clicked), value: stats.clickedCount },
    { label: t(($) => $.broadcasts.stats.unsubscribed), value: stats.unsubscribedCount },
    { label: t(($) => $.broadcasts.stats.failed), value: stats.failedCount },
  ]
}

export function BroadcastReportPage() {
  const { t } = useTranslation()
  const { slug, broadcastId } = broadcastsReportRoute.useParams()

  const getQuery = useQuery(
    siteBroadcastsGetOptions({ path: { workspaceSlug: slug, id: broadcastId } }),
  )

  if (getQuery.isLoading) return <Loader />

  if (getQuery.isError || !getQuery.data) {
    return (
      <ApiErrorAlert
        error={getQuery.error}
        title={t(($) => $.alerts.broadcastLoadErrorTitle)}
        fallback={t(($) => $.alerts.broadcastLoadErrorTitle)}
      />
    )
  }

  const broadcast = getQuery.data

  return (
    <Stack>
      <Group justify="space-between">
        <Title order={4}>{broadcast.name}</Title>
        <Badge color={STATUS_COLORS[broadcast.status]} variant="light" size="lg">
          {t(($) => $.broadcasts.status[broadcast.status])}
        </Badge>
      </Group>
      <Text c="dimmed">{broadcast.subject}</Text>

      <SimpleGrid cols={{ base: 2, sm: 3 }}>
        {statEntries(t, broadcast.stats).map((entry) => (
          <StatCard key={entry.label} label={entry.label} value={entry.value} />
        ))}
      </SimpleGrid>
    </Stack>
  )
}
