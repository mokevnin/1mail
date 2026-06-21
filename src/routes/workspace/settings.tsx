import {
  Alert,
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
import { useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteWorkspacesListOptions,
  siteWorkspacesListQueryKey,
  siteWorkspacesUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteWorkspaceResource } from '../../generated/site/types.gen.ts'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'

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

// TrackingSection shows the embed snippet customers paste into their site. The
// host is derived from the current origin; the collect key identifies the workspace.
function TrackingSection({ collectKey }: { collectKey: string }) {
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
          <TrackingSection collectKey={workspace.collectKey} />
          <Card withBorder>
            <Title order={4} mb="xs">
              {t(($) => $.settings.apiKeysTitle)}
            </Title>
            <Text c="dimmed" size="sm">
              {t(($) => $.settings.apiKeysComingSoon)}
            </Text>
          </Card>
        </>
      ) : null}
    </Stack>
  )
}
