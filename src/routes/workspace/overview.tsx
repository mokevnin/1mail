import { Button, Card, SimpleGrid, Stack, Text, Title } from '@mantine/core'
import { useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteContactsListOptions,
  siteWorkspacesListOptions,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { activityRoute, contactsRoute, settingsRoute } from '../../router.tsx'

// OverviewPage is the workspace landing: headline metrics and quick links into
// the main sections. Metrics are intentionally sparse for now (more to come).
export function OverviewPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = useParams({ strict: false })

  const workspacesQuery = useQuery(siteWorkspacesListOptions())
  const workspace = workspacesQuery.data?.find((w) => w.slug === slug)

  const contactsQuery = useQuery({
    ...siteContactsListOptions({ path: { workspaceSlug: slug ?? '' }, query: { pageSize: 1 } }),
    enabled: Boolean(slug),
  })
  const contactsCount = contactsQuery.data?.totalItems ?? 0

  if (!slug) {
    return null
  }

  return (
    <Stack>
      <Title order={2}>{workspace?.name ?? t(($) => $.overview.title)}</Title>

      <SimpleGrid cols={{ base: 1, sm: 3 }}>
        <Card withBorder>
          <Text c="dimmed" size="sm">
            {t(($) => $.overview.contactsCount)}
          </Text>
          <Title order={3}>{contactsCount}</Title>
          <Button
            variant="light"
            mt="sm"
            onClick={() => navigate({ to: contactsRoute.to, params: { slug } })}
          >
            {t(($) => $.overview.viewContacts)}
          </Button>
        </Card>

        <Card withBorder>
          <Text c="dimmed" size="sm">
            {t(($) => $.nav.activity)}
          </Text>
          <Button
            variant="light"
            mt="sm"
            onClick={() => navigate({ to: activityRoute.to, params: { slug } })}
          >
            {t(($) => $.overview.viewActivity)}
          </Button>
        </Card>

        <Card withBorder>
          <Text c="dimmed" size="sm">
            {t(($) => $.nav.settings)}
          </Text>
          <Button
            variant="light"
            mt="sm"
            onClick={() => navigate({ to: settingsRoute.to, params: { slug } })}
          >
            {t(($) => $.overview.viewSettings)}
          </Button>
        </Card>
      </SimpleGrid>

      <Text c="dimmed" size="sm">
        {t(($) => $.overview.comingSoon)}
      </Text>
    </Stack>
  )
}
