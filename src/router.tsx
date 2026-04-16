import { createRootRoute, createRoute, createRouter, redirect } from '@tanstack/react-router'
import App from './App.tsx'
import { ContactCreatePage } from './routes/contacts/create.tsx'
import { ContactEditPage } from './routes/contacts/edit.tsx'
import { ContactsListPage } from './routes/contacts/list.tsx'

const rootRoute = createRootRoute({
  component: App,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/contacts' })
  },
})

const contactsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/contacts',
  component: ContactsListPage,
})

const contactsCreateRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/contacts/new',
  component: ContactCreatePage,
})

const contactsEditRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/contacts/$contactId/edit',
  component: ContactEditPage,
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  contactsRoute,
  contactsCreateRoute,
  contactsEditRoute,
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
