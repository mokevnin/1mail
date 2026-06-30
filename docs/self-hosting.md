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
`production` when self-hosting).

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `DATABASE_URL` | **yes** | — | PostgreSQL connection string (`postgres://user:pass@host:5432/db?sslmode=…`). |
| `JWT_SECRET` | **yes in prod** | — | Signing secret for auth tokens. The server refuses to boot outside `development`/`test` if this is empty. |
| `ENCRYPTION_KEY` | **yes** | — | Base64 Tink keyset used to encrypt stored provider credentials. Generate one with `go run ./cmd/genkey` (or `1mail`-side tooling). Boot fails if missing. |
| `APP_URL` | no | `http://localhost:3000` | Public base URL — used when issuing auth tokens and building tracking/unsubscribe links. Set to your real origin. |
| `PORT` | no | `3000` | HTTP listen port. |
| `AUTO_MIGRATE` | no | `false` | Apply embedded migrations on startup. Convenient for single-replica; see below. |
| `CORS_ORIGINS` | no | — | Comma/space-separated list of allowed CORS origins. |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` / `SMTP_FROM` | no | `SMTP_PORT=1025` | Outbound email over SMTP. |
| `SYSTEM_EMAIL_PROVIDER` | no | `smtp` | Platform (system) email provider: `smtp` or `ses`. |
| `SYSTEM_EMAIL_FROM` | no | `noreply@1mail.localhost` | From address for platform mail (e.g. welcome emails). |
| `SES_REGION` / `SES_ACCESS_KEY_ID` / `SES_SECRET_ACCESS_KEY` | no | — | Amazon SES credentials when using the `ses` provider. |
| `COLLECT_SITE_KEY` | no | — | Tracker ingestion key. |
| `BOOTSTRAP_TOKEN` | no | — | External-API bootstrap token. |

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

> The binary tracks applied migrations in its own `schema_migrations` table. Don't point
> it at a database previously managed by the Atlas dev-CLI flow (which uses
> `atlas_schema_revisions`).

## Run it

### Docker

```sh
docker run -p 3000:3000 \
  -e APP_ENV=production \
  -e DATABASE_URL="postgres://user:pass@db:5432/1mail?sslmode=disable" \
  -e JWT_SECRET="$(openssl rand -hex 32)" \
  -e ENCRYPTION_KEY="<base64 tink keyset>" \
  -e APP_URL="https://mail.example.com" \
  -e AUTO_MIGRATE=true \
  ghcr.io/mokevnin/1mail:latest
```

The image declares a `HEALTHCHECK` against `/healthz`, so `docker ps` reports `healthy`
once the process is serving.

### docker-compose

```yaml
services:
  app:
    image: ghcr.io/mokevnin/1mail:latest
    ports: ["3000:3000"]
    environment:
      APP_ENV: production
      DATABASE_URL: postgres://postgres:postgres@db:5432/1mail?sslmode=disable
      JWT_SECRET: change-me
      ENCRYPTION_KEY: <base64 tink keyset>
      APP_URL: https://mail.example.com
      AUTO_MIGRATE: "true"
    depends_on:
      db:
        condition: service_healthy
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: 1mail
      POSTGRES_PASSWORD: postgres
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 3s
      retries: 10
    volumes: ["pgdata:/var/lib/postgresql/data"]

volumes:
  pgdata:
```

### Binary

```sh
export APP_ENV=production
export DATABASE_URL="postgres://user:pass@host:5432/1mail?sslmode=disable"
export JWT_SECRET="$(openssl rand -hex 32)"
export ENCRYPTION_KEY="<base64 tink keyset>"
export APP_URL="https://mail.example.com"

./1mail migrate   # apply migrations (or set AUTO_MIGRATE=true)
./1mail           # start the server
```

## Health checks

| Endpoint | Purpose | Behaviour |
| --- | --- | --- |
| `GET /healthz` | Liveness | Always `200 {"status":"ok"}` while the process serves; touches no dependency. |
| `GET /readyz` | Readiness | Pings the database; `200` when reachable, `503` (`application/problem+json`) otherwise. |

Wire `/healthz` to liveness and `/readyz` to readiness probes (Kubernetes, load
balancers, the Docker `HEALTHCHECK`).
