import { initTracking } from '@1mail/analytics'
import { Group, Select } from '@mantine/core'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { AppNavbar } from '../components/AppNavbar.tsx'
import { UserMenu } from '../components/UserMenu.tsx'
import { siteWorkspacesListOptions } from '../generated/site/@tanstack/react-query.gen.ts'
import type { SiteWorkspaceResource } from '../generated/site/types.gen.ts'
import { overviewRoute, workspaceRoute } from '../router.tsx'
import { DashboardShell } from './DashboardShell.tsx'

// WorkspaceSwitcher changes the active workspace by navigating to its
// /w/{slug} route.
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
          navigate({ to: overviewRoute.to, params: { slug: value } })
        }
      }}
      w={220}
    />
  )
}

// WorkspaceLayout is the shell for workspace-scoped pages (overview, contacts,
// activity, workspace settings): workspace sidebar + switcher.
export function WorkspaceLayout() {
  const { slug } = workspaceRoute.useParams()
  const workspacesQuery = useQuery(siteWorkspacesListOptions())
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
    <DashboardShell
      sidebar={<AppNavbar slug={slug} />}
      headerRight={
        <Group gap="sm">
          <WorkspaceSwitcher slug={slug} workspaces={workspaces} />
          <UserMenu slug={slug} />
        </Group>
      }
    />
  )
}
