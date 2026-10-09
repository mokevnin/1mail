# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

1mail is an **open-core marketing automation platform** (with a planned SaaS offering).
Go backend + React/Vite frontend in a single repo. The data model is workspace-scoped
(multi-tenant): contacts, events, api tokens, and tracking entities all belong to a
`workspace`.

## Codegen pipeline (read this first)

API contracts are **one-directional**: TypeSpec → OpenAPI → generated Go + TS. Never
hand-edit anything under `openapi/`, `gen/`, `ent/` (except `ent/schema/`),
`src/generated/` / `packages/analytics/src/generated/`, or the `*_gen.go` files in the
`internal/api/{site,external}/resources` packages — regenerate instead.

```
typespec/{site,external,collect}   ──tsp compile──▶  openapi/*.openapi.json
  openapi/site      ──ogen──▶ gen/site       (Go server)   ──openapi-ts──▶ src/generated/site (TS client + react-query + zod)
  openapi/external  ──ogen──▶ gen/external    (Go server)
  openapi/collect   ──ogen──▶ gen/collect     (Go server)  ──openapi-ts──▶ packages/analytics/src/generated/collect (types only)
ent/schema/*.go     ──entc──▶ ent/*           (Go ORM)
ent + gen/{site,external}  ──goverter──▶ internal/api/{site,external}/resources/converter_gen.go
```

- **goverter** maps ent entities → ogen resource DTOs. The `Converter` interface and its
  `// goverter:extend` helpers (ids, timestamps, optionals) are hand-written in
  each package's `resources.go`; the impl (`ConverterImpl`) is generated. Adding a DTO
  field that has no source mapping fails generation — that completeness check is the point.

- `make generate` — full cycle: typespec → openapi → backend (ent + ogen) → frontend → i18n types → format.
- `make generate-typespec` / `make generate-backend` / `make generate-openapi` — partial regens.
- The README claims echo + oapi-codegen; that is **outdated**. The HTTP stack is **ogen**
  (`gen/*` are ogen servers, wired in `internal/server/server.go`).

### What is generated vs hand-written

Generated files carry a header (`Code generated … DO NOT EDIT`, or `// @ts-nocheck` on the
frontend) and are flagged `linguist-generated` in `.gitattributes` (GitHub collapses them in
diffs). Generated code lives in dedicated dirs — `internal/` is otherwise hand-written, its
only generated files being the two `converter_gen.go`.

| Path                                                      | Generator           | Regen via                | Hand-written source                                 |
| --------------------------------------------------------- | ------------------- | ------------------------ | --------------------------------------------------- |
| `ent/` (except `schema/`, `entc.go`, `generate.go`)       | ent/entc            | `make generate-backend`  | `ent/schema/*.go`, `ent/entc.go`                    |
| `gen/{site,external,collect}/`                            | ogen                | `make generate-backend`  | — (from `openapi/`)                                 |
| `internal/api/{site,external}/resources/converter_gen.go` | goverter            | `make generate-backend`  | sibling `resources.go` (interface + extend helpers) |
| `openapi/*.openapi.json`                                  | TypeSpec            | `make generate-typespec` | `typespec/{site,external,collect}/`                 |
| `src/generated/site/`                                     | @hey-api/openapi-ts | `make generate-openapi`  | — (from `openapi/`)                                 |
| `packages/analytics/src/generated/collect/`               | @hey-api/openapi-ts | `make generate-openapi`  | — (from `openapi/`)                                 |

## Common commands

Every recipe runs natively with the toolchain pinned in `.mise.toml` / `mise.lock`
(`mise install`); CI installs the same toolchain with `jdx/mise-action`. There are no dev
containers. The dev stack (Postgres, Mailpit, backend, Vite, Caddy) is a set of **mise
daemons** (`[daemons]` in `.mise.toml`, experimental).

```sh
make setup          # mise install, deps, start Postgres, create dev/test/atlas DBs + migrate + seed
make install        # mise install + pnpm install + go mod download
make dev            # mise daemons start caddy — full dev stack (https://1mail.localhost)
make dev-down       # mise daemons stop
make test           # creates test DB, then `go test -p 1 ./...`
make check          # tsc + oxlint + oxfmt --check + knip + golangci-lint + govulncheck + gitleaks + jactionlint + zizmor
make check-fix      # oxlint --fix + oxfmt + tsp format + go fmt
```

`DATABASE_URL` and `PG*` come from the `db` daemon (mise `postgres` preset, auto port from
15432), so run `make`/`go` through a mise-activated shell (or `mise exec -- make …`). Never
set `DATABASE_URL` in `.env`: explicit values override the daemon's. `TEST_DB_URL` and
`ATLAS_DB_URL` derive from `PGHOST`/`PGPORT`/`PGUSER`. Recipes that need a specific env
(`APP_ENV=test`, a test/atlas DB URL) pass it as a plain env prefix.

Run a single Go test:

```sh
mise exec -- sh -c 'APP_ENV=test DATABASE_URL=$TEST_DB_URL go test ./internal/api/site -run TestSiteContactsRequireAuth'
```

Frontend tests: `make test-watch`.

## Database & migrations

- ORM is **ent**; schemas live in `ent/schema/*.go`, generated code in `ent/`.
- Migrations are managed by **Atlas** (`atlas.hcl`, dir `migrations/`), run natively, diffed from the ent schema: `make db-generate name=<desc>` then `make db-migrate`.
  Atlas reads its target/dev DB URLs from the environment (`DATABASE_URL` / `ATLAS_DEV_URL`),
  using a scratch `atlas_dev` database on the `db` daemon instead of `docker://`.
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
  `internal/jobs` uses river (Postgres-backed queue). Email via `internal/messaging`
  (per-workspace providers: smtp/ses). Both providers share `messaging.BuildMIME`
  (wneessen/go-mail) — smtp sends the `*mail.Msg` directly, ses serializes it to raw
  bytes for SES `SendRawEmail`.
- `cmd/server` (HTTP server), `cmd/db` (create/drop DBs), `cmd/seed` (seed data).
- The tracker snippet (`/t.js`) is the `@1mail/analytics` IIFE bundle, built and embedded:
  `make build-tracker` copies `packages/analytics/dist/t.js` into `internal/server/assets/`.

### Go test harness

`internal/testhelper.Setup(t)` is the standard fixture. It migrates schema + loads
committed YAML fixtures from `fixtures/` **once per process**, then each test runs inside
a **go-txdb** transaction that rolls back on cleanup (full isolation, DB stays at fixture
state). Tests drive the real server in-memory via the **typed ogen client** + an injecting
transport (`env.Transport(headers)`) — no sockets. See `internal/api/site/contacts_test.go`.

**Build test scenarios on the committed fixtures — don't fabricate the primary
entities inline.** Reference fixture rows by their known IDs (e.g. draft broadcast 100,
sent broadcast 200 with recipient rows 1000+, anchor contacts 1–3, anchor segments 1–2);
query counts that vary with the dataset dynamically instead of hardcoding them. If a
scenario isn't covered, **add a fixture row** rather than a `client.X.Create()` in the
test. Anchor rows flagged "DO NOT change" in the YAML are referenced across the suite —
leave them. txdb rolls back each test, so mutating a fixture row inside a test is fine.
Exception: incidental one-off records (e.g. a suppression to trip a specific edge) may be
created inline when no fixture expresses them.

## Frontend architecture

- React 19 + Vite + Mantine + TanStack Router/Query. Entry `src/main.tsx`, routes in
  `src/router.tsx` / `src/routes/`. Note the README mentions oRPC, but the site client is
  actually the generated `@hey-api/openapi-ts` fetch client + react-query hooks in
  `src/generated/site/` (consume these, don't hand-write fetch calls).
- Route auth guard hits `/site/contacts?pageSize=1` and redirects to `/login` on 401.
- i18n via i18next; `locales/`, types generated by `make generate-i18n-types`.
- Lint is **oxlint** (all plugins, type-aware via oxlint-tsgolint; `.oxlintrc.json`) and format is **oxfmt** (`.oxfmtrc.json`; also formats Markdown/YAML/JSON). Single quotes, no semicolons, generated dirs ignored. **Never disable a lint rule** (no `off`, no `oxlint-disable`/ignore comments) — fix the code; configure a rule's own options only when it has them.
- Unused deps/files/exports are caught by **knip** (`knip.json`); secrets by **gitleaks** (`.gitleaks.toml`); workflows by **jactionlint** + **zizmor**; Go vulns by **govulncheck**.
  Type-check is **tsc** (TypeScript 7, the native compiler).

## Dev environment

`make dev` starts the mise daemons (`.mise.toml`) behind Caddy with HTTPS at
**https://1mail.localhost** (run `caddy trust` once). The external API is also exposed at
**https://api.1mail.localhost** — Caddy rewrites `/*` → `/api/*` to the same backend, so
the subdomain root mirrors the binary's `/api` path (RudderStack-style edge; the binary
stays path-based). In prod the ingress in front of the binary does the same rewrite for
`api.onemail.dev`. Daemons: `db` (mise `postgres` preset), `mailpit` (SMTP UI at :8025),
`backend` (real Go server under air on `:3300`, hot reload), `frontend` (Vite on `:5173`),
`caddy`. Inspect with `mise daemons ls|logs|status`. Migrations run via atlas
(`make db-migrate`); the backend does not self-migrate. Dev defaults (JWT secret, dev
`ENCRYPTION_KEY`, SMTP) live in the `[env]` table of `.mise.toml`; personal overrides go in
the gitignored `.env` (read by the app) or `.mise.local.toml`.

## Conventions

- Commit directly to `main` (no feature branches).
- Commit messages follow **Conventional Commits** (`feat:`, `fix:`, `chore:`, `docs:`,
  `refactor:`, `ci:` …) — release-please uses them for versioning/changelog.
- After changing TypeSpec or `ent/schema`, run `make generate` and commit the generated output.
- **No custom CSS anywhere in the repo.** Style the frontend exclusively through Mantine — components, style
  props (`p`, `c`, `w`, responsive object syntax), the color system, the theme, and the
  configured breakpoints. Do not add custom `.css`/CSS-module files, inline `style={{…}}`,
  hardcoded colors, or CSS-in-JS (styled-components/emotion). The only CSS imports are the
  library stylesheets in `src/main.tsx`.
- **Theming & responsiveness.** Light and dark color schemes are supported via
  `MantineProvider` (`defaultColorScheme="auto"`) + `ColorSchemeScript` in `index.html`;
  the UI must be responsive using Mantine primitives (responsive props,
  `visibleFrom`/`hiddenFrom`, `AppShell` breakpoints, `SimpleGrid` cols) — never custom CSS.

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`mokevnin/1mail`), via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five-label vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` at the repo root plus `docs/adr/`. See `docs/agents/domain.md`.
