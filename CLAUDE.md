# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

1mail is an **open-core marketing automation platform** (with a planned SaaS offering).
Go backend + React/Vite frontend in a single repo. The data model is workspace-scoped
(multi-tenant): contacts, events, api tokens, and tracking entities all belong to a
`workspace`.

## Codegen pipeline (read this first)

API contracts are **one-directional**: TypeSpec → OpenAPI → generated Go + TS. Never
hand-edit anything under `openapi/`, `gen/`, `ent/` (except `ent/schema/`), or
`src/generated/` / `packages/analytics/src/generated/` — regenerate instead.

```
typespec/{site,external,collect}   ──tsp compile──▶  openapi/*.openapi.json
  openapi/site      ──ogen──▶ gen/site       (Go server)   ──openapi-ts──▶ src/generated/site (TS client + react-query + zod)
  openapi/external  ──ogen──▶ gen/external    (Go server)
  openapi/collect   ──ogen──▶ gen/collect     (Go server)  ──openapi-ts──▶ packages/analytics/src/generated/collect (types only)
ent/schema/*.go     ──entc──▶ ent/*           (Go ORM)
```

- `make generate` — full cycle: typespec → openapi → backend (ent + ogen) → frontend → i18n types → format.
- `make generate-typespec` / `make generate-backend` / `make generate-openapi` — partial regens.
- The README claims echo + oapi-codegen; that is **outdated**. The HTTP stack is **ogen**
  (`gen/*` are ogen servers, wired in `internal/server/server.go`).

## Common commands

Every recipe runs its toolchain **inside the dev containers** (docker-compose.yml), so a
host with only Docker works. The runner vars are overridable — CI runs everything natively
with `make <target> RUN_FE= RUN_GO= RUN_GO_DB=`.

```sh
make setup          # build images, install deps, create dev/test/atlas DBs + migrate
make install        # pnpm install + go mod download (both in containers)
make dev            # docker compose up — full dev stack
make test           # creates test DB, then `go test -p 1 ./...`
make check          # tsgo --noEmit + biome check + golangci-lint
make check-fix      # biome --write + tsp format + go fmt
```

Runner vars (Makefile): `RUN_FE` (node tools, frontend image), `RUN_GO` (go tools, no DB),
`RUN_GO_DB` (go tools that need Postgres — starts the `db` service). Recipes that need a
specific env (`APP_ENV=test`, a test/atlas DB URL) wrap the command in `sh -c '…'` so the
assignment works both in-container and on the CI-native override path.

Run a single Go test:
```sh
docker compose run --rm backend go test ./internal/api/site -run TestSiteContactsRequireAuth
```
Frontend tests: `make test-watch`.

## Database & migrations

- ORM is **ent**; schemas live in `ent/schema/*.go`, generated code in `ent/`.
- Migrations are managed by **Atlas** (`atlas.hcl`, dir `migrations/`), run in the backend
  container, diffed from the ent schema: `make db-generate name=<desc>` then `make db-migrate`.
  Atlas reads its target/dev DB URLs from the environment (`DATABASE_URL` / `ATLAS_DEV_URL`),
  using a scratch `atlas_dev` database on the compose `db` service instead of `docker://`.
- `make db-reset` / `db-reset-test` to rebuild local DBs.
- Test DB is separate (`APP_ENV=test`); tests create schema via `ent` `Schema.Create`, not Atlas.

## Backend architecture

- **DI**: `samber/do` container. `internal/app/app.go` `register()` wires every singleton
  (config, sql.DB, ent client, email sender, pubsub, the `http.Handler`). Add new
  dependencies there via `do.Provide`.
- **HTTP**: `internal/server/server.go` `New()` mounts the three ogen servers plus
  go-pkgz/auth onto a stdlib `http.ServeMux`, then wraps it with hand-rolled middleware
  (recoverer, requestID, timeout, CORS). Errors render as RFC 7807 `application/problem+json`.
- **Three API surfaces**, each with its own TypeSpec spec, ogen server, handler package,
  and auth scheme:
  - `/site/*` — frontend SPA API. Auth: **JWT cookie** (issued by go-pkgz/auth). Handlers in `internal/api/site`.
  - `/api/*` — external/public API. Auth: **Bearer api-token** (workspace-scoped). Handlers in `internal/api/external`.
  - `/collect/*` — tracking ingestion from customer sites. Auth: **x-collect-key** header. Handlers in `internal/api/collect`.
- Auth security handlers live in `internal/api/auth/auth.go`. External requests carry a
  `*TokenAuth` (workspace id + scopes) in context — **scope all external queries by workspace**.
- **Async**: `internal/pubsub` (watermill over Postgres) — handlers registered in
  `pubsub.RegisterHandlers`, router run in a goroutine from `cmd/server/main.go`.
  `internal/jobs` uses river (Postgres-backed queue). Email via `internal/email` (go-mail).
- `cmd/server` (HTTP server), `cmd/db` (create/drop DBs), `cmd/seed` (seed data).
- The tracker snippet (`/t.js`) is the `@1mail/analytics` IIFE bundle, built and embedded:
  `make build-tracker` copies `packages/analytics/dist/t.js` into `internal/server/assets/`.

### Go test harness

`internal/testhelper.Setup(t)` is the standard fixture. It migrates schema + loads
committed YAML fixtures from `fixtures/` **once per process**, then each test runs inside
a **go-txdb** transaction that rolls back on cleanup (full isolation, DB stays at fixture
state). Tests drive the real server in-memory via the **typed ogen client** + an injecting
transport (`env.Transport(headers)`) — no sockets. See `internal/api/site/contacts_test.go`.

## Frontend architecture

- React 19 + Vite + Mantine + TanStack Router/Query. Entry `src/main.tsx`, routes in
  `src/router.tsx` / `src/routes/`. Note the README mentions oRPC, but the site client is
  actually the generated `@hey-api/openapi-ts` fetch client + react-query hooks in
  `src/generated/site/` (consume these, don't hand-write fetch calls).
- Route auth guard hits `/site/contacts?pageSize=1` and redirects to `/login` on 401.
- i18n via i18next; `locales/`, types generated by `make generate-i18n-types`.
- Lint/format is **biome** (single quotes, no semicolons, generated dirs ignored).
  Type-check is **tsgo** (`@typescript/native-preview`, not stock tsc).

## Dev environment

`make dev` runs docker compose (`docker-compose.yml`) behind Caddy with HTTPS at
**https://1mail.localhost**. The external API is also exposed at
**https://api.1mail.localhost** — Caddy rewrites `/*` → `/api/*` to the same backend, so
the subdomain root mirrors the binary's `/api` path (RudderStack-style edge; the binary
stays path-based). In prod the ingress in front of the binary does the same rewrite for
`api.onemail.dev`. Services: caddy, frontend (vite), backend, postgres, mailpit
(SMTP UI at :8025). Deps and the Go module cache are bind-mounted from the host
(`node_modules` in the repo, the module cache under `./.cache/go-mod`) so host editor LSPs
resolve imports — install via `make install` (runs in the containers). The compose
`backend` runs the real Go server (`Dockerfile.backend.dev`, `golang:1.26-alpine` + air,
plus golangci-lint + atlas for the tooling recipes) on `:3300` with sources bind-mounted
and hot reload via air. Migrations run via atlas in the container (`make db-migrate`); the
backend service does not self-migrate.

## Conventions

- Commit directly to `main` (no feature branches).
- Commit messages follow **Conventional Commits** (`feat:`, `fix:`, `chore:`, `docs:`,
  `refactor:`, `ci:` …) — release-please uses them for versioning/changelog.
- After changing TypeSpec or `ent/schema`, run `make generate` and commit the generated output.
- **No custom CSS.** Style the frontend exclusively through Mantine — components, style
  props (`p`, `c`, `w`, responsive object syntax), the color system, the theme, and the
  configured breakpoints. Do not add custom `.css`/CSS-module files, inline `style={{…}}`,
  hardcoded colors, or CSS-in-JS (styled-components/emotion). The only CSS imports are the
  library stylesheets in `src/main.tsx`.
- **Theming & responsiveness.** Light and dark color schemes are supported via
  `MantineProvider` (`defaultColorScheme="auto"`) + `ColorSchemeScript` in `index.html`;
  the UI must be responsive using Mantine primitives (responsive props,
  `visibleFrom`/`hiddenFrom`, `AppShell` breakpoints, `SimpleGrid` cols) — never custom CSS.
