# Self-hosting 1mail

1mail ships as a **single self-contained artifact** — one Go binary (or one Docker image)
with the React SPA, the tracker snippet (`/t.js`), and the database migrations all
embedded. The only external dependency at runtime is **PostgreSQL** (the background queue
and pub/sub both ride on Postgres — no Redis, no object storage, no separate worker).

- **PostgreSQL:** 14+ recommended.
- Outbound email via SMTP or Amazon SES.

## Configuration

Configuration is read from environment variables (and, if present, `.env` files next to
the binary). `APP_ENV` selects the environment (`development` by default; set it to
`production` when self-hosting — the published Docker images already default to `production`).

| Variable                                                            | Required        | Default                   | Description                                                                                                                                                                                                          |
| ------------------------------------------------------------------- | --------------- | ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `DATABASE_URL`                                                      | **yes**         | —                         | PostgreSQL connection string (`postgres://user:pass@host:5432/db?sslmode=require`). Use `sslmode=verify-full` (with `sslrootcert`) to also verify the server certificate.                                            |
| `JWT_SECRET`                                                        | **yes in prod** | —                         | Signing secret for auth tokens. Outside `development`/`test` the server refuses to boot if it is empty, shorter than 32 characters, or a known placeholder. Generate one with `openssl rand -hex 32`.                |
| `ENCRYPTION_KEY`                                                    | **yes**         | —                         | Base64 Tink keyset used to encrypt stored provider credentials. Generate one with `go run ./cmd/genkey` (or `1mail`-side tooling). Boot fails if missing.                                                            |
| `APP_URL`                                                           | no              | `http://localhost:3000`   | Public base URL — used when issuing auth tokens and building tracking/unsubscribe links. Set to your real origin.                                                                                                    |
| `PORT`                                                              | no              | `3000`                    | HTTP listen port.                                                                                                                                                                                                    |
| `METRICS_ADDR`                                                      | no              | — (off)                   | `host:port` for the opt-in Prometheus listener. Must differ from `PORT`; see [Prometheus metrics](#prometheus-metrics).                                                                                              |
| `AUTO_MIGRATE`                                                      | no              | `false`                   | Apply embedded migrations on startup. Convenient for single-replica; see below.                                                                                                                                      |
| `CORS_ORIGINS`                                                      | no              | —                         | Comma/space-separated origins allowed credentialed CORS on the cookie-authenticated API (`/site`, `/auth`). Empty means same-origin only, which is what the bundled SPA needs; the bearer-token APIs are unaffected. |
| `MAX_BODY_BYTES`                                                    | no              | `1048576`                 | Largest accepted request body (bytes) on every surface except `/collect`; larger bodies get `413`.                                                                                                                   |
| `COLLECT_MAX_BODY_BYTES`                                            | no              | `65536`                   | Largest accepted request body (bytes) on `/collect`.                                                                                                                                                                 |
| `DB_MAX_OPEN_CONNS`                                                 | no              | `15`                      | Maximum open connections in the `database/sql` pool (API, ent, event bus).                                                                                                                                           |
| `DB_MAX_IDLE_CONNS`                                                 | no              | `DB_MAX_OPEN_CONNS`       | Maximum idle connections kept in that pool. Must not exceed `DB_MAX_OPEN_CONNS`.                                                                                                                                     |
| `DB_CONN_MAX_LIFETIME`                                              | no              | `30m`                     | Maximum lifetime of a pooled connection (Go duration).                                                                                                                                                               |
| `PGX_MAX_CONNS`                                                     | no              | `25`                      | Maximum connections in the river (job queue) pool. Must cover river's MaxWorkers sum (20) plus LISTEN and runtime services.                                                                                          |
| `OUTBOX_RETENTION_FLOOR_DAYS`                                       | no              | `7`                       | Minimum age (days) before a domain-event outbox row that every consumer has processed is pruned. The prune job runs every 10 minutes.                                                                                |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` / `SMTP_FROM` | no              | `SMTP_PORT=1025`          | Outbound email over SMTP.                                                                                                                                                                                            |
| `SYSTEM_EMAIL_PROVIDER`                                             | no              | `smtp`                    | Platform (system) email provider: `smtp` or `ses`.                                                                                                                                                                   |
| `SYSTEM_EMAIL_FROM`                                                 | no              | `noreply@1mail.localhost` | From address for platform mail (e.g. welcome emails).                                                                                                                                                                |
| `SES_REGION` / `SES_ACCESS_KEY_ID` / `SES_SECRET_ACCESS_KEY`        | no              | —                         | Amazon SES credentials when using the `ses` provider.                                                                                                                                                                |
| `COLLECT_SITE_KEY`                                                  | no              | —                         | Tracker ingestion key.                                                                                                                                                                                               |
| `BOOTSTRAP_TOKEN`                                                   | no              | —                         | External-API bootstrap token.                                                                                                                                                                                        |

## Database migrations

Migrations are embedded in the binary. You apply them one of two ways:

- **Single replica:** set `AUTO_MIGRATE=true` and the binary migrates on startup.
- **Multiple replicas:** do **not** use `AUTO_MIGRATE` (replicas would race). Run the
  migration step once before rolling out the new version:

  ```sh
  ./1mail migrate      # applies pending migrations and exits
  ```

  Use it as a pre-deploy job or a Kubernetes init container, then start the servers
  without `AUTO_MIGRATE`.

Before upgrading, take a database backup; see [Upgrading](./operations/upgrading) for the
full procedure and [Backup and restore](./operations/backup) for what to protect.

> The binary tracks applied migrations (via goose) in its own `goose_db_version` table. Don't point
> it at a database previously managed by the Atlas dev-CLI flow (which uses
> `atlas_schema_revisions`).

## Run it

### Docker

```sh
docker run -p 3000:3000 \
  -e APP_ENV=production \
  -e DATABASE_URL="postgres://user:pass@db:5432/1mail?sslmode=require" \
  -e JWT_SECRET="$(openssl rand -hex 32)" \
  -e ENCRYPTION_KEY="<base64 tink keyset>" \
  -e APP_URL="https://mail.example.com" \
  -e AUTO_MIGRATE=true \
  ghcr.io/mokevnin/1mail:latest
```

The image declares a `HEALTHCHECK` against `/healthz`, so `docker ps` reports `healthy`
once the process is serving.

### Binary

```sh
export APP_ENV=production
export DATABASE_URL="postgres://user:pass@host:5432/1mail?sslmode=require"
export JWT_SECRET="$(openssl rand -hex 32)"
export ENCRYPTION_KEY="<base64 tink keyset>"
export APP_URL="https://mail.example.com"

./1mail migrate   # apply migrations (or set AUTO_MIGRATE=true)
./1mail           # start the server
```

## Health checks

| Endpoint       | Purpose   | Behaviour                                                                               |
| -------------- | --------- | --------------------------------------------------------------------------------------- |
| `GET /healthz` | Liveness  | Always `200 {"status":"ok"}` while the process serves; touches no dependency.           |
| `GET /readyz`  | Readiness | Pings the database; `200` when reachable, `503` (`application/problem+json`) otherwise. |

Wire `/healthz` to liveness and `/readyz` to readiness probes (Kubernetes, load
balancers, the Docker `HEALTHCHECK`).

## Prometheus metrics

Metrics are off by default and are never served on the public port. Set `METRICS_ADDR` to
start a dedicated listener that serves `GET /metrics` (Prometheus exposition) and nothing
else. There is no token and no allowlist: the network boundary is the control (ADR 0018).

- Single host: `METRICS_ADDR=127.0.0.1:9090` so only local scrapers can reach it.
- Container: `METRICS_ADDR=0.0.0.0:9090`, and publish that port only to your Prometheus
  (a network policy or an internal network), never through the public ingress.

A `METRICS_ADDR` using the same port as `PORT`, or a malformed value, fails startup, as does
a metrics address that is already in use. The Dockerfiles need no change (`EXPOSE 3000` and
the `HEALTHCHECK` stay). Scrape example:

```yaml
scrape_configs:
  - job_name: 1mail
    static_configs:
      - targets: ['1mail:9090']
```

OTLP push (`OTEL_EXPORTER_OTLP_*`) is independent of this and unchanged.

## Operations

Sizing, [backup and restore](./operations/backup) and [upgrading](./operations/upgrading)
(including PostgreSQL major upgrades) are covered in the [Operations](./operations/) section.
