import { AppShell, Group, Title } from '@mantine/core'
import { Outlet } from '@tanstack/react-router'
import type { ReactNode } from 'react'

// DashboardShell is the common authenticated chrome shared by the
// workspace-scoped and account (workspace-independent) layouts: the brand
// header with a right-side slot, a left sidebar slot, and the routed main area.
export function DashboardShell({
  sidebar,
  headerRight,
}: {
  sidebar: ReactNode
  headerRight?: ReactNode
}) {
  return (
    <AppShell header={{ height: 64 }} navbar={{ width: 260, breakpoint: 'sm' }} padding="md">
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Title order={3}>1mail</Title>
          {headerRight}
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">{sidebar}</AppShell.Navbar>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  )
}
