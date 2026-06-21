import { AppShell, Burger, Group, Title } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { Outlet, useLocation } from '@tanstack/react-router'
import { type ReactNode, useEffect } from 'react'
import { ThemeToggle } from '../components/ThemeToggle.tsx'

// DashboardShell is the common authenticated chrome shared by the
// workspace-scoped and account (workspace-independent) layouts: the brand
// header with a right-side slot, a left sidebar slot, and the routed main area.
// On narrow viewports the sidebar collapses behind a burger; it reopens per
// navigation and auto-closes whenever the route changes.
export function DashboardShell({
  sidebar,
  headerRight,
}: {
  sidebar: ReactNode
  headerRight?: ReactNode
}) {
  const [opened, { toggle, close }] = useDisclosure(false)
  const location = useLocation()

  // Close the mobile drawer after navigating so the overlay never lingers.
  useEffect(() => {
    close()
  }, [location.pathname, close])

  return (
    <AppShell
      header={{ height: 64 }}
      navbar={{ width: 260, breakpoint: 'sm', collapsed: { mobile: !opened } }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between" wrap="nowrap">
          <Group gap="sm" wrap="nowrap">
            <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
            <Title order={3}>1mail</Title>
          </Group>
          <Group gap="sm" wrap="nowrap">
            {headerRight}
            <ThemeToggle />
          </Group>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">{sidebar}</AppShell.Navbar>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  )
}
