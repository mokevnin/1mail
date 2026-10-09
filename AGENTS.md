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
`internal/api/{site,external}/resources` packages and `internal/fixtures/catalog_gen.go` — regenerate instead.

```
typespec/{site,external,collect}   ──tsp compile──▶  openapi/*.openapi.json
  openapi/site      ──ogen──▶ gen/site       (Go server)   ──openapi-ts──▶ src/generated/site (TS client + react-query + zod)
  openapi/external  ──ogen──▶ gen/external    (Go server)
  openapi/collect   ──ogen──▶ gen/collect     (Go server)  ──openapi-ts──▶ packages/analytics/src/generated/collect (types only)
ent/schema/*.go     ──entc──▶ ent/*           (Go ORM)
ent/schema/*.go + ent/template/scoped*.tmpl  ──entc──▶ ent/scoped.go, ent/scoped_registry.go  (scoped client: `client.Scoped(ws)`)
ent + gen/{site,external}  ──goverter──▶ internal/api/{site,external}/resources/converter_gen.go
fixtures/*.yml (`# fixture: Name` rows)  ──cmd/fixturegen──▶ internal/fixtures/catalog_gen.go (named test constants)
```

- **goverter** maps ent entities → ogen resource DTOs. The `Converter` interface and its
  `// goverter:extend` helpers (ids, timestamps, optionals) are hand-written in
  each package's `resources.go`; the impl (`ConverterImpl`) is generated. Adding a DTO
  field that has no source mapping fails generation — that completeness check is the point.

- `mise run generate` — full cycle: typespec → openapi → backend (ent + ogen) → fixture catalog → frontend → i18n types → format.
- `mise run generate:typespec` / `mise run generate:backend` / `mise run generate:openapi` — partial regens.
- The README claims echo + oapi-codegen; that is **outdated**. The HTTP stack is **ogen**
  (`gen/*` are ogen servers, wired in `internal/server/server.go`).

### What is generated vs hand-written

Generated files carry a header (`Code generated … DO NOT EDIT`, or `// @ts-nocheck` on the
frontend) and are flagged `linguist-generated` in `.gitattributes` (GitHub collapses them in
diffs). Generated code lives in dedicated dirs — `internal/` is otherwise hand-written, its
only generated files being the two `converter_gen.go` and `internal/fixtures/catalog_gen.go`.

## Common commands

Every task runs natively with the toolchain pinned in `.mise.toml` / `mise.lock` (Go and
Node track `latest`); CI installs the same toolchain with `jdx/mise-action` and runs the same
tasks. There are no dev containers and no Makefile: tasks live in `mise/tasks.toml` (included
from `.mise.toml`); `mise tasks` lists them. The dev stack (Postgres, Mailpit, backend, Vite,
Caddy) is a set of **mise daemons** (`[daemons]` in `.mise.toml`, experimental), and tasks that
need Postgres declare `daemons = ["db"]`, so they start it themselves.

```sh
mise run setup          # deps, start Postgres, create dev/test/atlas DBs, migrate, seed
mise run dev            # mise daemons start caddy — full dev stack (https://1mail.localhost)
mise run dev:down       # mise daemons stop
mise run test           # creates the test DB, then `go test -p 1 ./...`
mise run check          # tsc + oxlint + oxfmt --check + knip + golangci-lint + govulncheck + gitleaks + jactionlint + zizmor
mise run fix            # i18n extract + oxlint --fix + tsp format + oxfmt + go fmt
mise run generate       # typespec -> openapi -> backend -> frontend -> i18n types -> format
```

`DATABASE_URL` and `PG*` come from the `db` daemon (mise `postgres` preset, auto port from
15432). Never set `DATABASE_URL` in `.env`: explicit values override the daemon's.
`TEST_DB_URL` and `ATLAS_DEV_URL` derive from `PGHOST`/`PGPORT`/`PGUSER` (set them in the
environment to use an external Postgres). Several tasks in one command need `:::` between
them (`mise run check:fe ::: check:deps`); otherwise the extra names become arguments.

Run a single Go test (arguments after `--` go to `go test`; the default is `./...`):

```sh
mise run test -- ./internal/api/site -run TestSiteContactsRequireAuth
```

Frontend tests: `mise run test:watch`.

## Database & migrations

- ORM is **ent**; schemas live in `ent/schema/*.go`, generated code in `ent/`.
- Migrations are managed by **Atlas** (`atlas.hcl`, dir `migrations/`), run natively, diffed from the ent schema: `mise run db:generate name=<desc>` then `mise run db:migrate`.
  Atlas reads its target/dev DB URLs from the environment (`DATABASE_URL` / `ATLAS_DEV_URL`),
  using a scratch `atlas_dev` database on the `db` daemon instead of `docker://`.
- `mise run db:reset` / `db:reset-test` to rebuild local DBs.
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
  `*TokenAuth` (scopes + the Workspace's `*ent.Scoped`) in context — query through
  `auth.TokenScoped(ctx)` (see "Workspace scoping" below).

### Workspace scoping: the generated scoped client

Every Workspace-owned entity (one with `WorkspaceMixin`) is reached through
`client.Scoped(ws)` (`*ent.Scoped`), generated by the entc templates in
`ent/template/scoped*.tmpl` (ADR 0017). Its `Query`/`Create`/`Get`/`UpdateOneID`/`Update`/
`Delete`/`DeleteOneID`/`CreateBulk`/upsert wrappers apply the Workspace themselves; `Create`
has no `SetWorkspaceID`; a reference to another Workspace's row fails with
`ent.ErrNotInWorkspace` (422). A new Workspace-owned entity is covered on the next
`mise run generate`; nothing is hand-written per entity. `Scoped.Workspace(ctx)` loads the
tenant row itself (the Workspace is the tenant root, so it has no wrapper).

- **Handlers and domain modules hold no `*ent.Client` and no bare `int64` workspace id.**
  They take a `*ent.Scoped` (modules: an explicit `s *ent.Scoped` parameter) and never call
  `client.Scoped(<int64>)`. Tests use `env.DB.Scoped(fixtures.AcmeID)`.
- **`*ent.Scoped` is built only in a closed list of places:**
  1. site: `accounts.Accounts.Scope`, behind `Handlers.scopedFor` / `scopedWithRoleFor`
     (`internal/api/site/handlers.go`), from the user's Membership on `/w/{slug}`;
  2. external (and MCP, which rides the same token auth): `ExternalSecurityHandler.HandleBearerAuth`
     in `internal/api/auth/auth.go`, read with `auth.TokenScoped(ctx)`;
  3. collect: `CollectSecurityHandler` in `internal/api/auth/auth.go`, from the collect key;
  4. jobs and event subscribers, right after the row that names the Workspace is loaded
     (`client.Scoped(run.WorkspaceID)` in `internal/jobs`, `events.Persist`/`Suppress`);
  5. transactions: `events.Bus.WithinScopedTx`, and `tx.Scoped(...)` inside
     `accounts.AcceptInvitation`, re-scope a transaction's client to the same Workspace;
  6. the Workspace comes from a secret rather than a login: the SES hook ingest key
     (`internal/server/hooks_ses.go`), signed unsubscribe/confirm tokens
     (`internal/consent`), and `accounts.BootstrapScope` (the bootstrap token).
- **The raw `*ent.Client` is allowed only in:** `internal/accounts` (User, Membership,
  Workspace, invitation by token), `internal/api/auth` (credentials, token and key lookup),
  `internal/oauthserver`, `internal/service` (suspension, slug resolution), `internal/events`
  (the bus and its subscribers), `internal/jobs` (job entry points), `internal/server`
  (tracking by recipient id, provider hooks, composition) and the composition roots
  (`internal/app`, `internal/db`, `internal/testhelper`). Needing raw access anywhere else
  means a new small package in this list, not a field on `Handlers`.
- **Lint:** `forbidigo` (`.golangci.yml`) bans entity-level `entity.Update()` repo-wide,
  because a loaded entity still carries the raw client. Update by id instead:
  `s.Tag().UpdateOneID(id)`. There are no exclusions.
- **Async**: `internal/pubsub` (watermill over Postgres) — handlers registered in
  `pubsub.RegisterHandlers`, router run in a goroutine from `cmd/server/main.go`.
  `internal/jobs` uses river (Postgres-backed queue). Email via `internal/messaging`
  (per-workspace providers: smtp/ses). Both providers share `messaging.BuildMIME`
  (wneessen/go-mail) — smtp sends the `*mail.Msg` directly, ses serializes it to raw
  bytes for SES `SendRawEmail`.
- The tracker snippet (`/t.js`) is the `@1mail/analytics` IIFE bundle, built and embedded:
  `mise run build:tracker` copies `packages/analytics/dist/t.js` into `internal/server/assets/`.

## Frontend architecture

- The README mentions oRPC, but the site client is
  actually the generated `@hey-api/openapi-ts` fetch client + react-query hooks in
  `src/generated/site/` (consume these, don't hand-write fetch calls).
- Route auth guard hits `/site/workspaces` (`fetchWorkspaces` in `src/router.tsx`) and redirects to `/login` on 401.
- i18n via i18next; `locales/`, types generated by `mise run generate:i18n-types`.
- Lint is **oxlint** (all plugins, type-aware via oxlint-tsgolint; `oxlint.config.ts`) and format is **oxfmt** (`oxfmt.config.ts`; also formats Markdown/YAML/JSON). Single quotes, no semicolons, generated dirs ignored. **Never disable a lint rule** (no `off`, no `oxlint-disable`/ignore comments) — fix the code; configure a rule's own options only when it has them.
- Unused deps/files/exports are caught by **knip** (`knip.ts`); secrets by **gitleaks** (`.gitleaks.toml`); workflows by **jactionlint** + **zizmor**; Go vulns by **govulncheck**.
  Type-check is **tsc** (TypeScript 7, the native compiler).

## Dev environment

`mise run dev` starts the mise daemons (`.mise.toml`) behind Caddy with HTTPS at
**https://1mail.localhost** (run `caddy trust` once). The external API is also exposed at
**https://api.1mail.localhost** — Caddy rewrites `/*` → `/api/*` to the same backend, so
the subdomain root mirrors the binary's `/api` path (RudderStack-style edge; the binary
stays path-based). In prod the ingress in front of the binary does the same rewrite for
`api.onemail.dev`. Daemons: `db` (mise `postgres` preset), `mailpit` (SMTP UI at :8025),
`backend` (real Go server under air on `:3300`, hot reload), `frontend` (Vite on `:5173`),
`caddy`. Inspect with `mise daemons ls|logs|status`. Migrations run via atlas
(`mise run db:migrate`); the backend does not self-migrate. Dev defaults (JWT secret, dev
`ENCRYPTION_KEY`, SMTP) live in the `[env]` table of `.mise.toml`; personal overrides go in
the gitignored `.env` (read by the app) or `.mise.local.toml`.

## Conventions

- Commit messages follow **Conventional Commits** (`feat:`, `fix:`, `chore:`, `docs:`,
  `refactor:`, `ci:` …) — release-please uses them for versioning/changelog.
- After changing TypeSpec or `ent/schema`, run `mise run generate` and commit the generated output.
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

### Advisor before /to-spec

Before running the `to-spec` skill, call the `advisor` tool first and take its advice into account. The skill is vendored, so this rule lives here rather than in the skill.
