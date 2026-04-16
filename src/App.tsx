import { AppShell, Group, Title } from '@mantine/core'
import { Outlet } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

export default function App() {
  const { t } = useTranslation()

  return (
    <AppShell header={{ height: 64 }} padding="md">
      <AppShell.Header>
        <Group h="100%" px="md">
          <Title order={3}>{t(($) => $.contacts.title)}</Title>
        </Group>
      </AppShell.Header>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  )
}
