import { AppShell, Group, Title } from '@mantine/core'
import { Outlet, useLocation } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { trackPageView } from './tracking.ts'

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

export default function App() {
  const { t } = useTranslation()

  return (
    <AppShell header={{ height: 64 }} padding="md">
      <RouteTracking />
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
