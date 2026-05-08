import { createRootRoute, createRoute, createRouter, redirect } from '@tanstack/react-router'
import App from './App.tsx'
import { LoginPage } from './routes/auth/login.tsx'
import { RegisterPage } from './routes/auth/register.tsx'
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

const authGuard = async () => {
  const res = await fetch('/site/contacts?pageSize=1', { credentials: 'include' })
  if (res.status === 401) {
    throw redirect({ to: '/login' })
  }
}

export const contactsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/contacts',
  beforeLoad: authGuard,
  component: ContactsListPage,
})

export const contactsCreateRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/contacts/new',
  beforeLoad: authGuard,
  component: ContactCreatePage,
})

export const contactsEditRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/contacts/$contactId/edit',
  beforeLoad: authGuard,
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
