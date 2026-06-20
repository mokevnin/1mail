import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  redirect,
} from '@tanstack/react-router'
import App from './App.tsx'
import { siteWorkspacesList } from './generated/site/sdk.gen.ts'
import type { SiteWorkspaceResource } from './generated/site/types.gen.ts'
import { LoginPage } from './routes/auth/login.tsx'
import { RegisterPage } from './routes/auth/register.tsx'
import { ContactCreatePage } from './routes/contacts/create.tsx'
import { ContactEditPage } from './routes/contacts/edit.tsx'
import { ContactsListPage } from './routes/contacts/list.tsx'

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
    throw redirect({ to: contactsRoute.to, params: { slug: first.slug } })
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
      throw redirect({ to: contactsRoute.to, params: { slug: fallback.slug } })
    }
    return { workspaces }
  },
  component: Outlet,
})

export const contactsRoute = createRoute({
  getParentRoute: () => workspaceRoute,
  path: 'contacts',
  component: ContactsListPage,
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

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  registerRoute,
  workspaceRoute.addChildren([contactsRoute, contactsCreateRoute, contactsEditRoute]),
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
