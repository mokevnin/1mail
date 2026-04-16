# AGENTS.md

## Quick Start
- Install deps: `pnpm install` (or `make setup`).
- Backend dev server: `make dev-backend` (Fastify on `app.ts`, default `http://localhost:3000`).
- Frontend dev server: `make dev-frontend` (Vite).
- Do not rely on `make dev`/`Procfile` right now: they reference `overmind` + `pnpm run dev:frontend|dev:backend`, but `package.json` has no scripts.

## Verification (match CI)
- Main gate is `make check && make test`.
- `make check` runs `npx tsgo --noEmit` then `npx biome check .`.
- `make test` runs `pnpm exec vitest run`.
- Focused tests:
  - Single file: `pnpm exec vitest run test/routes/contacts.test.ts`
  - Single test: `pnpm exec vitest run test/routes/contacts.test.ts -t "contacts CRUD works with pglite"`

## Runtime + DB Rules
- Runtime DB driver is resolved in `db/runtime.ts`:
  - default: `pglite` outside production
  - production default: `postgres`
  - explicit override: `DB_DRIVER=pglite|postgres`
- Postgres runtime requires `DATABASE_URL`.
- Drizzle CLI (`drizzle.config.ts`) always requires `DATABASE_URL` for `make db-generate` / `make db-migrate`, even if app runtime uses `pglite`.
- `pglite` runtime auto-applies migrations from `drizzle/` in `plugins/drizzle.ts`.

## API Contract + Generated Code
- Source of truth for API shape is TypeSpec in `typespec/`.
- Generate OpenAPI: `make generate-typespec` -> writes `openapi/openapi.json`.
- Generate typed handlers/types: `pnpm exec openapi-ts -f openapi-ts.config.ts` -> writes `generated/handlers/*`.
- Do not hand-edit generated files under `generated/`.

## Architecture Notes (only what is easy to miss)
- Backend entrypoint is `app.ts`; Fastify autoloads plugins from `plugins/` and external HTTP API routes from `server/http/`.
- Implemented external API module is `server/http/contacts/index.ts`; `server/http/segments/index.ts` and `server/http/broadcasts/index.ts` are placeholders that throw `not implemented`.
- Internal frontend API lives in `server/trpc/`; frontend page routes live in `src/routes/`.
- Contacts handlers are typed from generated OpenAPI types (`RouteHandlers`) and use `toFastifySchema(...)` from `lib/openapi.ts`.
- Route files under `server/http/<name>/index.ts` are mounted with the folder prefix. Inside them, register relative paths (`/`, `/:id`) to avoid doubled paths.

## Repo-Specific Quirks
- `.npmrc` sets `node-options="--experimental-strip-types"`, so TS files are executed directly by node-based tooling here.
- `README.md` is Fastify boilerplate and does not describe the current workflow.

## Rules

- Test only http endpoints (server/http). Check only status code.
