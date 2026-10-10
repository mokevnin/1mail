# 1mail

Open-core marketing automation you can run yourself. 1mail helps a business know its
audience and talk to it at the right moment: it collects who your contacts are and what
they do on your site or in your product, groups them into segments, and sends them email —
one-off broadcasts, automated sequences triggered by their behavior, and transactional
messages.

It looks after the unglamorous part too: unsubscribes, bounces and spam complaints are
honored automatically, so you never write to someone who asked you to stop, and your sender
reputation stays intact. Everything lives in your own workspace on your own infrastructure,
so the customer data stays yours. The goal is a self-hostable alternative to tools like
Drip, without the lock-in.

![The 1mail workspace overview: email engagement, contacts and automations](docs/public/screenshot.png)

**Documentation:** https://mokevnin.github.io/1mail/

## Stack

A Go backend and a React/Vite frontend in a single repo. The data model is
workspace-scoped (multi-tenant): contacts, events, API tokens, and tracking entities all
belong to a `workspace`.

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
mise run generate          # full cycle: typespec → openapi → backend (ent + ogen) → frontend → i18n types → format
mise run generate:typespec # typespec → openapi only
mise run generate:backend  # openapi → Go only (ent + ogen)
mise run generate:openapi  # openapi → TS client only
```

After changing TypeSpec or `ent/schema`, run `mise run generate` and commit the generated output.

## Development

**The only prerequisite is [mise](https://mise.jdx.dev).** It installs and pins the whole
toolchain from `.mise.toml` / `mise.lock` (Go, Node, pnpm, golangci-lint, atlas, air,
Caddy, Mailpit, gitleaks, jactionlint, zizmor, hk, …), runs the dev stack as daemons and
installs the git hooks.

```sh
mise install # install the pinned toolchain (also installs the git hooks)
mise run setup   # install deps, start Postgres, create dev/test/atlas DBs, migrate, seed
mise run dev     # start the whole dev stack (mise daemons)
mise run test    # creates test DB, then `go test -p 1 ./...`
mise run check   # tsc, oxlint, oxfmt --check, knip, golangci-lint, govulncheck, gitleaks, jactionlint, zizmor
mise run generate # regenerate TypeSpec → OpenAPI → Go + TS
```

CI installs the same toolchain with `jdx/mise-action` and runs the same mise tasks (`mise tasks` lists them all).

## Deployment (development)

The local stack is a set of [mise daemons](https://mise.jdx.dev) (`[daemons]` in
`.mise.toml`) behind [Caddy](https://caddyserver.com/) with HTTPS.

1. Install the toolchain, install deps, create the dev/test/atlas databases, migrate:

   ```sh
   mise install
   mise run setup
   ```

   No `.env` is required: `.mise.toml` supplies the dev defaults and the `db` daemon
   supplies `DATABASE_URL`. Do **not** set `DATABASE_URL` in `.env` (real env vars win
   over it). Personal overrides go in the gitignored `.env` (read by the app) or
   `.mise.local.toml` (read by mise).

2. Trust Caddy's local CA once, so the browser accepts `https://1mail.localhost`:

   ```sh
   caddy trust
   ```

3. Start the stack:

   ```sh
   mise run dev        # mise daemons start caddy (starts everything it depends on)
   mise run dev:down   # stop it
   mise daemons logs backend   # follow a daemon's output; `mise daemons ls` shows status
   ```

The entry point is **https://1mail.localhost** (Caddy terminates TLS with its internal CA).
Daemons:

| Daemon   | URL / port                     | Notes                                                       |
| -------- | ------------------------------ | ----------------------------------------------------------- |
| caddy    | `:443`                         | TLS + routing, see `Caddyfile`                              |
| frontend | `:5173` (Vite)                 | proxied by Caddy                                            |
| backend  | `:3300`                        | proxied by Caddy under `/site`, `/api`, `/collect`, `/auth` |
| db       | `127.0.0.1:15432` (`mise env`) | mise `postgres` preset, dev DB `1mail_development`          |
| mailpit  | http://localhost:8025          | captured outbound email (SMTP UI on `:1025`)                |

> The backend runs the real Go server under [air](https://github.com/air-verse/air) for
> hot reload. Migrations run via Atlas (`mise run db:migrate`); the dev backend itself does not
> self-migrate. On Linux, binding `:443` needs `net.ipv4.ip_unprivileged_port_start=0`.
> The old Grafana/OTLP dev container is gone: leave `OTEL_EXPORTER_OTLP_ENDPOINT` unset
> (the dev stack serves Prometheus metrics on `127.0.0.1:9090/metrics` via the opt-in
> `METRICS_ADDR`; see [ADR 0018](docs/adr/0018-metrics-opt-in-internal-listener.md)) or point it
> at your own collector.

## Deployment (production)

The server ships as a **self-contained static binary**: the React SPA, the `t.js` tracker,
and the database migrations are all embedded (`go:embed`, built with `-tags embed_spa`).
No Node.js, no Atlas CLI, and no extra runtime dependencies are needed.

### Docker image

Published to **`ghcr.io/mokevnin/1mail`** (multi-arch, linux amd64/arm64) on every release,
tagged with the version and `latest`.

```sh
docker run -p 3000:3000 \
  -e APP_ENV=production \
  -e DATABASE_URL="postgres://user:pass@host:5432/1mail?sslmode=require" \
  -e APP_URL="https://example.com" \
  -e JWT_SECRET="$(openssl rand -hex 32)" \
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
mise run build    # → bin/1mail (build:tracker + build:spa + go build -tags embed_spa)
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

> The production binary tracks applied migrations (via goose) in its own `goose_db_version` table. The
> dev Atlas CLI flow (`mise run db:migrate`) uses Atlas's `atlas_schema_revisions` table — never
> point `AUTO_MIGRATE` at a database previously managed by the Atlas CLI dev flow.

### Configuration

Configuration is read from the environment (and, if present, `.env` files).

| Variable                                                            | Default                  | Description                                                                                                                    |
| ------------------------------------------------------------------- | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------ |
| `DATABASE_URL`                                                      | — (**required**)         | PostgreSQL connection string                                                                                                   |
| `PORT`                                                              | `3000`                   | HTTP listen port                                                                                                               |
| `APP_URL`                                                           | `http://localhost:3000`  | Public base URL (auth token issuance)                                                                                          |
| `AUTO_MIGRATE`                                                      | `false`                  | Apply embedded migrations on startup                                                                                           |
| `JWT_SECRET`                                                        | — (**required in prod**) | JWT signing secret; required outside development                                                                               |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` / `SMTP_FROM` | `SMTP_PORT=1025`         | Outbound email                                                                                                                 |
| `CORS_ORIGINS`                                                      | —                        | Origins allowed credentialed CORS on the cookie API (`/site`, `/auth`); empty = same-origin only                               |
| `MAX_BODY_BYTES` / `COLLECT_MAX_BODY_BYTES`                         | `1048576` / `65536`      | Largest accepted request body in bytes (`/collect` has its own cap); larger bodies get `413`                                   |
| `OUTBOX_RETENTION_FLOOR_DAYS`                                       | `7`                      | Minimum age in days before a consumed domain-event outbox row is pruned (ADR 0019)                                             |
| `EVENTS_RETENTION_DAYS`                                             | `400`                    | Age in days after which analytical Events are deleted daily at 03:00 UTC; `0` disables. Evidentiary Events are kept (ADR 0019) |

`COLLECT_SITE_KEY` and `BOOTSTRAP_TOKEN` are also recognized (tracker ingestion key and
external-API bootstrap token).

For deploying 1mail yourself (Docker or single binary, full env-var reference, migrations,
health checks), see [`docs/self-hosting.md`](docs/self-hosting.md).

## License

1mail is **open-core**. The core is licensed under the GNU AGPL-3.0
([`LICENSE`](LICENSE)); the Enterprise features under [`ee/`](ee/) are commercial and
source-available ([`ee/LICENSE`](ee/LICENSE)). See [`LICENSING.md`](LICENSING.md) for the
boundary.

## Security

To report a vulnerability, see [`SECURITY.md`](SECURITY.md).
