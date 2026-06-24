import {
  Alert,
  Anchor,
  Badge,
  Button,
  Card,
  Code,
  CopyButton,
  Group,
  Loader,
  Stack,
  Text,
  TextInput,
  Title,
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteEventsListOptions,
  siteWorkspacesListOptions,
  siteWorkspacesListQueryKey,
  siteWorkspacesUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteWorkspaceResource } from '../../generated/site/types.gen.ts'
import { activityRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import { ApiKeysSection } from './ApiKeysSection.tsx'
import { IntegrationsSection } from './IntegrationsSection.tsx'

// GeneralSection is split out so the rename form seeds its initial value from the
// loaded workspace without an effect.
function GeneralSection({ workspace }: { workspace: SiteWorkspaceResource }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const form = useForm<{ name: string }>({ initialValues: { name: workspace.name } })

  const updateMutation = useMutation({
    ...siteWorkspacesUpdateMutation(),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: siteWorkspacesListQueryKey() })
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.settings.saved),
      })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.settings.errorTitle),
        message: getApiErrorMessage(
          error,
          t(($) => $.settings.errorTitle),
        ),
      })
    },
  })

  return (
    <Card withBorder>
      <Title order={4} mb="sm">
        {t(($) => $.settings.generalTitle)}
      </Title>
      <form
        onSubmit={form.onSubmit((values) =>
          updateMutation.mutate({
            path: { slug: workspace.slug },
            body: { name: values.name.trim() },
          }),
        )}
      >
        <Stack maw={420}>
          <TextInput
            label={t(($) => $.settings.nameLabel)}
            required
            {...form.getInputProps('name')}
          />
          <Group justify="flex-end">
            <Button type="submit" loading={updateMutation.isPending}>
              {t(($) => $.actions.save)}
            </Button>
          </Group>
        </Stack>
      </form>
    </Card>
  )
}

// InstallStatus polls the events feed and tells the user whether their tracker
// is connected yet. Accurate only because the dashboard no longer self-tracks
// into the workspace (see App.tsx) — every event here comes from their own site.
function InstallStatus({ slug }: { slug: string }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const statusQuery = useQuery({
    ...siteEventsListOptions({
      path: { workspaceSlug: slug },
      query: { page: 1, pageSize: 1 },
    }),
    refetchInterval: 5000,
  })

  const totalItems = statusQuery.data?.totalItems ?? 0
  const received = totalItems > 0

  return (
    <Group mt="md" gap="sm">
      <Text fw={600}>{t(($) => $.settings.installStatusTitle)}:</Text>
      {received ? (
        <Badge color="teal" variant="light">
          {t(($) => $.settings.installStatusReceived)} ({totalItems})
        </Badge>
      ) : (
        <Badge color="gray" variant="light">
          {t(($) => $.settings.installStatusWaiting)}
        </Badge>
      )}
      <Anchor
        component="button"
        type="button"
        size="sm"
        onClick={() => navigate({ to: activityRoute.to, params: { slug } })}
      >
        {t(($) => $.settings.viewActivity)}
      </Anchor>
    </Group>
  )
}

// TestEvent shows a copy-paste curl command that posts a real event through the
// /collect pipeline (RudderStack-style): the user runs it, then watches it land
// in the activity feed. No in-app injection, so test traffic stays explicit.
function TestEvent({ collectKey }: { collectKey: string }) {
  const { t } = useTranslation()
  const origin = window.location.origin
  const command = [
    `curl -X POST "${origin}/collect/events" \\`,
    `  -H "content-type: application/json" \\`,
    `  -H "x-collect-key: ${collectKey}" \\`,
    `  -d '{"events":[{"visitorId":"test","action":"test.event"}]}'`,
  ].join('\n')

  return (
    <Card withBorder>
      <Title order={4} mb="xs">
        {t(($) => $.settings.testEventTitle)}
      </Title>
      <Text c="dimmed" size="sm" mb="sm">
        {t(($) => $.settings.testEventDescription)}
      </Text>
      <Code block>{command}</Code>
      <CopyButton value={command}>
        {({ copied, copy }) => (
          <Button mt="sm" variant="light" onClick={copy}>
            {copied ? t(($) => $.settings.copied) : t(($) => $.settings.copy)}
          </Button>
        )}
      </CopyButton>
    </Card>
  )
}

// TrackingSection shows the embed snippet customers paste into their site. The
// host is derived from the current origin; the collect key identifies the workspace.
function TrackingSection({ collectKey, slug }: { collectKey: string; slug: string }) {
  const { t } = useTranslation()
  const origin = window.location.origin
  const snippet = `<script async src="${origin}/t.js" data-collect-key="${collectKey}" data-collect-url="${origin}"></script>`

  return (
    <Card withBorder>
      <Title order={4} mb="xs">
        {t(($) => $.settings.trackingTitle)}
      </Title>
      <Text c="dimmed" size="sm" mb="sm">
        {t(($) => $.settings.trackingDescription)}
      </Text>
      <Code block>{snippet}</Code>
      <CopyButton value={snippet}>
        {({ copied, copy }) => (
          <Button mt="sm" variant="light" onClick={copy}>
            {copied ? t(($) => $.settings.copied) : t(($) => $.settings.copy)}
          </Button>
        )}
      </CopyButton>
      <InstallStatus slug={slug} />
    </Card>
  )
}

export function SettingsPage() {
  const { t } = useTranslation()
  const { slug } = useParams({ strict: false })
  const workspacesQuery = useQuery(siteWorkspacesListOptions())
  const workspace = workspacesQuery.data?.find((w) => w.slug === slug)

  return (
    <Stack>
      <Title order={2}>{t(($) => $.settings.title)}</Title>

      {workspacesQuery.isError ? (
        <Alert color="red" title={t(($) => $.settings.loadErrorTitle)}>
          {t(($) => $.settings.loadErrorTitle)}
        </Alert>
      ) : null}

      {workspacesQuery.isLoading ? <Loader /> : null}

      {workspace ? (
        <>
          <GeneralSection workspace={workspace} />
          <TrackingSection collectKey={workspace.collectKey} slug={workspace.slug} />
          <TestEvent collectKey={workspace.collectKey} />
          <IntegrationsSection slug={workspace.slug} />
          <ApiKeysSection slug={workspace.slug} />
        </>
      ) : null}
    </Stack>
  )
}
