// Where renderWithRouter mounts a page: the route's own pattern (from the app's route refs, so
// a path change in router.tsx carries over) and the concrete entry with its `$param`s filled in.
export function routeMount(route: { fullPath: string }, params: Record<string, string> = {}) {
  const initialPath = route.fullPath.replaceAll(/\$(\w+)/g, (_, name: string) => {
    const value = params[name]
    if (value === undefined) throw new Error(`missing route param ${name}`)
    return encodeURIComponent(value)
  })
  return { path: route.fullPath, initialPath }
}
