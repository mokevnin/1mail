import { trackPageView } from '@1mail/analytics'
import { Outlet, useLocation } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

// RouteTracking emits a page view on every route/locale change. It runs for all
// routes (including unauthenticated ones); the analytics client no-ops until a
// workspace layout initializes it with a collect key.
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

// App is the root: cross-cutting concerns only. Layout chrome lives in the
// per-area layouts (WorkspaceLayout, AccountLayout); auth pages render bare.
export default function App() {
  return (
    <>
      <RouteTracking />
      <Outlet />
    </>
  )
}
