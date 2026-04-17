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
- External SDK API entrypoint: `typespec/external/main.tsp` -> `openapi/external.openapi.json`.
- Site API entrypoint: `typespec/site/main.tsp` -> `openapi/site.openapi.json`.
- Generate OpenAPI: `make generate-typespec` -> writes both OpenAPI files.
- Generate typed handlers/types: `pnpm exec openapi-ts -f openapi-ts.config.ts` -> writes `generated/handlers/*`.
- Do not hand-edit generated files under `generated/`.

## Architecture Notes (only what is easy to miss)
- Backend entrypoint is `app.ts`; Fastify autoloads support plugins from `plugins/`, browser collection routes from `server/collect/` under `/collect`, and external HTTP API routes from `server/http/`.
- Implemented external HTTP modules currently include `root`, `contacts`, `events`, `event-actions`, and `auth`; `server/http/segments/index.ts` and `server/http/broadcasts/index.ts` are placeholders that throw `not implemented`.
- Internal frontend API lives in `server/trpc/`; frontend page routes live in `src/routes/`.
- HTTP handlers are typed from generated OpenAPI types (`RouteHandlers`) and validate requests with generated Zod schemas from `generated/handlers/zod.gen.ts`.
- Route files under `server/http/<name>/index.ts` are mounted with the folder prefix. Inside them, register relative paths (`/`, `/:id`) to avoid doubled paths.
- `resources/` is outbound-only: use it for `entity -> resource(dto) -> client` mapping. Request normalization, input shaping, and DB insert/update conversion belong in `use-cases/`, `lib/`, or transport-local helpers, not in `resources/`. If reusable DB record converters become common, put them under `records/`; keep one-off use-case-specific converters near their use-case/lib until that pattern emerges.

## Repo-Specific Quirks
- `.npmrc` sets `node-options="--experimental-strip-types"`, so TS files are executed directly by node-based tooling here.
- `README.md` is Fastify boilerplate and does not describe the current workflow.
- `make build` is currently not usable as a verification step because `package.json` does not define a `build` script.

## Rules

- Test only http endpoints (server/http). Check only status code.
