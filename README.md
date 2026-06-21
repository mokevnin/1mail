# 1mail

Open-core marketing automation platform — a Go backend and a React/Vite frontend in a
single repo. The data model is workspace-scoped (multi-tenant): contacts, events, API
tokens, and tracking entities all belong to a `workspace`.

## Stack

- **Backend** — Go + [ogen](https://ogen.dev/) (HTTP servers in `gen/*`), ORM
  [ent](https://entgo.io/), queues [river](https://riverqueue.com/), pub/sub
  [watermill](https://watermill.io/), migrations via [Atlas](https://atlasgo.io/).
- **Frontend** — React 19 + Vite, [Mantine](https://mantine.dev/),
  [TanStack Router/Query](https://tanstack.com/), and the generated
  [`@hey-api/openapi-ts`](https://heyapi.dev/) client + react-query hooks.
- **API contracts** — [TypeSpec](https://typespec.io/) as the single source of truth.

## Code generation

The pipeline is one-directional: TypeSpec → OpenAPI → generated Go + TS. Never hand-edit
anything under `openapi/`, `gen/`, `ent/` (except `ent/schema/`), or `src/generated/` /
`packages/analytics/src/generated/` — regenerate instead.

```
typespec/{site,external,collect}   ──tsp compile──▶  openapi/*.openapi.json
  openapi/site      ──ogen──▶ gen/site       (Go server)  ──openapi-ts──▶ src/generated/site (TS client + react-query + zod)
  openapi/external  ──ogen──▶ gen/external   (Go server)
  openapi/collect   ──ogen──▶ gen/collect    (Go server)  ──openapi-ts──▶ packages/analytics/src/generated/collect (types only)
ent/schema/*.go     ──entc──▶ ent/*          (Go ORM)
```

```sh
make generate          # full cycle: typespec → openapi → backend (ent + ogen) → frontend → i18n types → format
make generate-typespec # typespec → openapi only
make generate-backend  # openapi → Go only (ent + ogen)
make generate-openapi  # openapi → TS client only
```

After changing TypeSpec or `ent/schema`, run `make generate` and commit the generated output.

## Development

**The only prerequisite is Docker + Docker Compose.** Every `make` target runs its
toolchain (Go, Node/pnpm, golangci-lint, atlas) inside the dev containers, so you don't
need any of them installed on the host.

```sh
make setup   # build images, install deps, create dev/test/atlas DBs + migrate
make dev     # docker compose up — full dev stack
make test    # creates test DB, then `go test -p 1 ./...`
make check   # tsgo --noEmit, biome check, golangci-lint
make generate # regenerate TypeSpec → OpenAPI → Go + TS
```

Dependencies and the Go module cache are bind-mounted to the host (`node_modules` in the
repo, the module cache under `./.cache/go-mod`). So if you *do* run a host editor, gopls
and the TS language server resolve imports — point gopls at the module cache with
`go env -w GOMODCACHE=$PWD/.cache/go-mod` (optional; only needed for host LSP).

> CI installs the toolchains natively and runs the same targets without containers by
> overriding the runner vars, e.g. `make check RUN_FE= RUN_GO= RUN_GO_DB=`.

## Deployment (development)

The local stack runs via Docker Compose behind [Caddy](https://caddyserver.com/) with HTTPS.

1. Build images, install deps, create the dev/test/atlas databases, and migrate:

   ```sh
   make setup
   ```

   No `.env` is required — the Compose `backend` service supplies `DATABASE_URL`,
   `JWT_SECRET`, SMTP, etc. (Copy `.env.sample` only if you want to run the backend on the
   host instead of in a container.)

2. Start the stack:

   ```sh
   make dev        # docker compose up
   make dev-down   # stop it
   ```

The entry point is **https://1mail.localhost** (Caddy terminates TLS with its internal CA).
Services:

| Service  | URL / port                  | Notes                          |
| -------- | --------------------------- | ------------------------------ |
| frontend | `:5173` (Vite)              | proxied by Caddy               |
| backend  | `:3300`                     | proxied by Caddy under `/site`, `/api`, `/collect`, `/auth` |
| postgres | `localhost:5432`            | dev DB `1mail_development`      |
| mailpit  | http://localhost:8025       | captured outbound email (SMTP UI) |

> The Compose `backend` service runs the real Go server (`Dockerfile.backend.dev`,
> `golang:1.26-alpine` + [air](https://github.com/air-verse/air) for hot reload) with
> sources bind-mounted from the host. The same image carries golangci-lint and the Atlas
> CLI, so it also backs the `make` tooling recipes. Migrations run via Atlas in the
> container (`make db-migrate`); the dev backend itself does not self-migrate.

## Deployment (production)

The server ships as a **self-contained static binary**: the React SPA, the `t.js` tracker,
and the database migrations are all embedded (`go:embed`, built with `-tags embed_spa`).
No Node.js, no Atlas CLI, and no extra runtime dependencies are needed.

### Docker image

Published to **`ghcr.io/mokevnin/1mail`** (multi-arch, linux amd64/arm64) on every release,
tagged with the version and `latest`.

```sh
docker run -p 3000:3000 \
  -e DATABASE_URL="postgres://user:pass@host:5432/1mail?sslmode=disable" \
  -e APP_URL="https://example.com" \
  -e JWT_SECRET="<a-strong-secret>" \
  -e AUTO_MIGRATE=true \
  ghcr.io/mokevnin/1mail:latest
```

To build from source instead, use the multi-stage `Dockerfile` (node build → Go build →
Alpine runtime): `docker build -t 1mail .`.

### Binary

Release archives (`1mail_<version>_<os>_<arch>.tar.gz`) are attached to each
[GitHub Release](https://github.com/mokevnin/1mail/releases) for linux and darwin
(amd64/arm64). To build locally:

```sh
make build        # → bin/1mail (build-tracker + build-spa + go build -tags embed_spa)
```

Run it:

```sh
./bin/1mail migrate   # apply pending migrations and exit
./bin/1mail           # start the server (listens on $PORT, default 3000)
./bin/1mail version   # print build metadata
```

### Migrations

Two options:

- **Separate step (recommended)** — run `1mail migrate` before starting the server (an init
  container, release job, or manual step). Safe for multi-replica deploys.
- **On startup** — set `AUTO_MIGRATE=true` and the binary applies pending migrations
  in-process before serving. Use only for single-replica deploys against a fresh database.

> The production binary tracks applied migrations in its own `schema_migrations` table. The
> dev Atlas CLI flow (`make db-migrate`) uses Atlas's `atlas_schema_revisions` table — never
> point `AUTO_MIGRATE` at a database previously managed by the Atlas CLI dev flow.

### Configuration

Configuration is read from the environment (and, if present, `.env` files).

| Variable                                          | Default                  | Description                                  |
| ------------------------------------------------- | ------------------------ | -------------------------------------------- |
| `DATABASE_URL`                                    | — (**required**)         | PostgreSQL connection string                 |
| `PORT`                                            | `3000`                   | HTTP listen port                             |
| `APP_URL`                                          | `http://localhost:3000`  | Public base URL (auth token issuance)        |
| `AUTO_MIGRATE`                                    | `false`                  | Apply embedded migrations on startup         |
| `JWT_SECRET`                                      | —                        | JWT signing secret (set a real one in prod)  |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` / `SMTP_FROM` | `SMTP_PORT=1025` | Outbound email             |
| `CORS_ORIGINS`                                    | —                        | Allowed CORS origins                         |

`COLLECT_SITE_KEY` and `BOOTSTRAP_TOKEN` are also recognized (tracker ingestion key and
external-API bootstrap token).
