import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { type Mock, vi } from 'vitest'
import { renderWithProviders } from './renderWithProviders.tsx'

type RenderWithRouterOptions = {
  // Memory-history entry the router starts on. The default carries a `slug`
  // param so components calling useParams({ strict: false }) see one, matching
  // the app's workspace-scoped routes.
  initialPath?: string
  // Route pattern the component is mounted under. Must line up with initialPath.
  path?: string
}

// renderWithRouter mounts `ui` inside a minimal in-memory TanStack router so
// components that call useNavigate/useParams render without a real route tree.
// Navigation is not actually performed — `router.navigate` is replaced with a
// spy the test can assert against (e.g. expect(navigate).toHaveBeenCalledWith).
export async function renderWithRouter(
  ui: ReactNode,
  { initialPath = '/w/test', path = '/w/$slug' }: RenderWithRouterOptions = {},
) {
  const rootRoute = createRootRoute()
  const route = createRoute({
    getParentRoute: () => rootRoute,
    path,
    component: () => <>{ui}</>,
  })
  const router = createRouter({
    routeTree: rootRoute.addChildren([route]),
    history: createMemoryHistory({ initialEntries: [initialPath] }),
  })

  // Typed as Mock so the inferred return type stays nameable (no leaked
  // @vitest/spy internals). navigate is swapped in so tests can assert on it.
  const navigate: Mock = vi.fn().mockResolvedValue(undefined)
  router.navigate = navigate as unknown as typeof router.navigate

  const screen = await renderWithProviders(<RouterProvider router={router} />)
  return { screen, navigate }
}
