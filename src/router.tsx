import { createRootRoute, createRoute, createRouter, redirect } from '@tanstack/react-router'
import App from './App.tsx'
import { siteWorkspacesList } from './generated/site/sdk.gen.ts'
import type { SiteWorkspaceResource } from './generated/site/types.gen.ts'
import { AccountLayout } from './layouts/AccountLayout.tsx'
import { WorkspaceLayout } from './layouts/WorkspaceLayout.tsx'
import { ProfilePage } from './routes/account/profile.tsx'
import { LoginPage } from './routes/auth/login.tsx'
import { RegisterPage } from './routes/auth/register.tsx'
import { BroadcastCreatePage } from './routes/broadcasts/create.tsx'
import { BroadcastEditPage } from './routes/broadcasts/edit.tsx'
import { BroadcastsListPage } from './routes/broadcasts/list.tsx'
import { BroadcastReportPage } from './routes/broadcasts/report.tsx'
import { ContactCreatePage } from './routes/contacts/create.tsx'
import { ContactEditPage } from './routes/contacts/edit.tsx'
import { ContactsListPage } from './routes/contacts/list.tsx'
import { SegmentCreatePage } from './routes/segments/create.tsx'
import { SegmentEditPage } from './routes/segments/edit.tsx'
import { SegmentsListPage } from './routes/segments/list.tsx'
import { TemplateCreatePage } from './routes/templates/create.tsx'
import { TemplateEditPage } from './routes/templates/edit.tsx'
import { TemplatesListPage } from './routes/templates/list.tsx'
import { ActivityPage } from './routes/workspace/activity.tsx'
import { OverviewPage } from './routes/workspace/overview.tsx'
import { SettingsPage } from './routes/workspace/settings.tsx'

// fetchWorkspaces loads the authenticated user's workspaces via the generated
// client, redirecting to /login on 401. Used by route guards to authenticate
// and resolve the active workspace slug.
async function fetchWorkspaces(): Promise<SiteWorkspaceResource[]> {
  const { data, response } = await siteWorkspacesList()
  if (response?.status === 401) {
    throw redirect({ to: loginRoute.to })
  }
  return data ?? []
}

const rootRoute = createRootRoute({
  component: App,
})

export const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: async () => {
    const workspaces = await fetchWorkspaces()
    const first = workspaces[0]
    if (!first) {
      throw redirect({ to: loginRoute.to })
    }
    throw redirect({ to: overviewRoute.to, params: { slug: first.slug } })
  },
})

// Workspace-scoped layout: authenticates and verifies the slug is one the user owns.
export const workspaceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/w/$slug',
  beforeLoad: async ({ params }) => {
    const workspaces = await fetchWorkspaces()
    const current = workspaces.find((w) => w.slug === params.slug)
    if (!current) {
      const fallback = workspaces[0]
      if (!fallback) {
        throw redirect({ to: loginRoute.to })
      }
      throw redirect({ to: overviewRoute.to, params: { slug: fallback.slug } })
    }
    return { workspaces }
  },
  component: WorkspaceLayout,
})

// Index route for /w/$slug — the workspace landing.
export const overviewRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: '/',
  component: OverviewPage,
})

export const contactsRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'contacts',
  component: ContactsListPage,
})

export const segmentsRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'segments',
  component: SegmentsListPage,
})

export const segmentsCreateRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'segments/new',
  component: SegmentCreatePage,
})

export const segmentsEditRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'segments/$segmentId/edit',
  component: SegmentEditPage,
})

export const broadcastsRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'broadcasts',
  component: BroadcastsListPage,
})

export const broadcastsCreateRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'broadcasts/new',
  component: BroadcastCreatePage,
})

export const broadcastsEditRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'broadcasts/$broadcastId/edit',
  component: BroadcastEditPage,
})

export const broadcastsReportRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'broadcasts/$broadcastId/report',
  component: BroadcastReportPage,
})

export const templatesRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'templates',
  component: TemplatesListPage,
})

export const templatesCreateRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'templates/new',
  component: TemplateCreatePage,
})

export const templatesEditRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'templates/$templateId/edit',
  component: TemplateEditPage,
})

export const activityRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'activity',
  component: ActivityPage,
})

export const settingsRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'settings',
  component: SettingsPage,
})

export const contactsCreateRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'contacts/new',
  component: ContactCreatePage,
})

export const contactsEditRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'contacts/$contactId/edit',
  component: ContactEditPage,
})

export const registerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/register',
  component: RegisterPage,
})

export const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginPage,
})

// Account layout: workspace-independent settings. Authenticates via the same
// workspace fetch (redirects to /login on 401) but renders its own shell.
export const accountRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/account',
  beforeLoad: async () => {
    await fetchWorkspaces()
  },
  component: AccountLayout,
})

export const profileRoute = createRoute({
  getParentRoute: () => accountRoute,
  path: '/',
  component: ProfilePage,
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  registerRoute,
  accountRoute.addChildren([profileRoute]),
  workspaceRoute.addChildren([
    overviewRoute,
    contactsRoute,
    contactsCreateRoute,
    contactsEditRoute,
    segmentsRoute,
    segmentsCreateRoute,
    segmentsEditRoute,
    broadcastsRoute,
    broadcastsCreateRoute,
    broadcastsEditRoute,
    broadcastsReportRoute,
    templatesRoute,
    templatesCreateRoute,
    templatesEditRoute,
    activityRoute,
    settingsRoute,
  ]),
])

export const router = createRouter({
  routeTree,
  defaultPreload: 'intent',
  defaultPreloadStaleTime: 0,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
