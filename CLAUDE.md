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

```sh
make setup          # install deps + create dev & test DBs + migrate
make install        # pnpm install (via container) + go mod download
make dev            # docker compose up — full dev stack
make test           # creates test DB, then `go test -p 1 ./...`
make check          # tsgo --noEmit + biome check + golangci-lint (what CI runs)
make check-fix      # biome --write + tsp format + go fmt
```

Run a single Go test:
```sh
go test ./internal/api/site -run TestSiteContactsRequireAuth
```
Frontend tests use vitest: `make test-watch` or `pnpm exec vitest`.

`make check` needs golangci-lint v2 (pinned in `.mise.toml`; `mise install` or `brew install golangci-lint`).

## Database & migrations

- ORM is **ent**; schemas live in `ent/schema/*.go`, generated code in `ent/`.
- Migrations are managed by **Atlas** (`atlas.hcl`, dir `migrations/`), diffed from the
  ent schema: `make db-generate name=<desc>` then `make db-migrate`.
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
**https://1mail.localhost**. Services: caddy, frontend (vite), backend, postgres, mailpit
(SMTP UI at :8025). Deps are bind-mounted from the host so editor LSPs resolve imports —
install via `make install` (runs pnpm inside the container). The compose `backend` is
currently a `traefik/whoami` placeholder during the Go rewrite; run the real Go server on
the host (`:3300`) or swap the service in.

## Conventions

- Commit directly to `main` (no feature branches).
- After changing TypeSpec or `ent/schema`, run `make generate` and commit the generated output.
