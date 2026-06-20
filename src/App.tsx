import { initTracking, trackPageView } from '@1mail/analytics'
import { AppShell, Group, Select, Title } from '@mantine/core'
import { useQuery } from '@tanstack/react-query'
import { Outlet, useLocation, useNavigate, useParams } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { siteWorkspacesListOptions } from './generated/site/@tanstack/react-query.gen.ts'
import type { SiteWorkspaceResource } from './generated/site/types.gen.ts'
import { contactsRoute } from './router.tsx'

function RouteTracking() {
  const location = useLocation()
  const { i18n } = useTranslation()

  useEffect(() => {
    trackPageView({
      path: location.pathname,
      url: window.location.href,
      title: document.title,
      referrer: document.referrer,
      locale: i18n.language,
    })
  }, [i18n.language, location.pathname])

  return null
}

// WorkspaceSwitcher changes the active workspace by navigating to its
// /w/{slug} route. Only rendered when a workspace is active.
function WorkspaceSwitcher({
  slug,
  workspaces,
}: {
  slug: string
  workspaces: SiteWorkspaceResource[]
}) {
  const navigate = useNavigate()
  const data = workspaces.map((w) => ({ value: w.slug, label: w.name }))

  return (
    <Select
      data={data}
      value={slug}
      allowDeselect={false}
      onChange={(value) => {
        if (value && value !== slug) {
          navigate({ to: contactsRoute.to, params: { slug: value } })
        }
      }}
      w={220}
    />
  )
}

export default function App() {
  const { t } = useTranslation()
  const { slug } = useParams({ strict: false })
  const workspacesQuery = useQuery({ ...siteWorkspacesListOptions(), enabled: !!slug })
  const workspaces = workspacesQuery.data ?? []
  const active = workspaces.find((w) => w.slug === slug)

  // Initialize dashboard self-tracking with the active workspace's collect key
  // (initTracking is idempotent — the first resolved workspace wins per session).
  useEffect(() => {
    if (active) {
      initTracking({
        collectKey: active.collectKey,
        baseUrl: import.meta.env.VITE_COLLECT_BASE_URL ?? '',
      })
    }
  }, [active])

  return (
    <AppShell header={{ height: 64 }} padding="md">
      <RouteTracking />
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Title order={3}>{t(($) => $.contacts.title)}</Title>
          {slug ? <WorkspaceSwitcher slug={slug} workspaces={workspaces} /> : null}
        </Group>
      </AppShell.Header>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  )
}
