# Sphericon hosted deployment: runbook

The hosted Sphericon deployment (DigitalOcean App Platform, Managed Postgres, DO DNS, AWS SES) is
described only by the Terraform in this directory. Why and what: [ADR 0028](../docs/adr/0028-saas-hosting-terraform-on-digitalocean-app-platform.md).
Follow the sections in order: prerequisites, credentials, plan, apply, registrar, smoke test;
destroy and recreate and troubleshooting come after.

**DigitalOcean MCP servers (`.mcp.json`: apps, databases, droplets) are for read-only inspection
(status, logs, listings), never for changes.** Every change goes through Terraform, otherwise the
state drifts and the next apply reverts it.

## 1. Prerequisites

- `mise install` provides `terraform`, `doctl`, the AWS CLI, `jq` and `shellcheck` (pinned in `mise.lock`).
- A DigitalOcean account, an AWS account (SES only), a GitHub account that can read the image on
  GHCR, and the registrar of `getsphericon.com`.
- `mise run check:infra` (fmt, `init -backend=false`, `validate`) needs no credentials; CI and the
  git hook run it.

## 2. Credentials: doctl is the single source

`doctl` is the only credential you set up by hand. `mise run infra` exports `DIGITALOCEAN_TOKEN`
from `doctl auth token` and reads every other secret from a DigitalOcean Secrets Manager container
(`sphericon-infra`, region `fra1`; override with `INFRA_SECRETS_NAME` and `INFRA_REGION`). Nothing
is stored in a file, nothing is passed as a command-line argument, and nothing is printed: values
travel through pipes and the process environment only. The doctl token needs the Secrets Manager
and Spaces scopes (a full-access token has them).

```sh
doctl auth init            # once: paste a DigitalOcean API token
mise run infra:bootstrap   # once: idempotent
mise run infra -- plan
```

`infra:bootstrap` creates, only when missing:

- the secret container with a fresh Spaces key (full access; the secret key exists only at
  creation, so it goes straight from `doctl` into the container), `ENCRYPTION_KEY`
  (`go run ./cmd/server genkey`), `JWT_SECRET` and `BOOTSTRAP_TOKEN`;
- the private state bucket `sphericon-tfstate` (override with `INFRA_STATE_BUCKET`; names are
  global across Spaces). `doctl` cannot create buckets, so the AWS CLI does it against the Spaces
  endpoint with the key it just stored, then verifies that the ACL grants nothing to `AllUsers`.

An existing container is never changed: `ENCRYPTION_KEY` is generated once and never rotated, and
it survives destroy and recreate of the environment (it protects stored provider credentials: a new
key makes them unreadable). If storing the new Spaces key fails, the key is deleted again.

`mise run infra -- <terraform args>` wraps Terraform in `infra/` (`init`, `plan`, `apply`,
`destroy`, `output`, ...): it initialises the backend by itself (bucket and key are non-secret
`-backend-config` values), passes `-var-file=production.tfvars` to `plan`, `destroy` and `apply`
(not to `apply <planfile>`), and fails with the list of container keys that are still missing.
Terraform's S3 backend only reads `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`, so the Spaces
key takes those names and the aws provider (SES) gets its own key through the
`aws_access_key_id` and `aws_secret_access_key` variables instead.

### Secrets you enter once

Not derivable, so they are stored once in the same container, each with
`mise run infra:secret NAME` (prompts without echo, or reads one line from stdin):

| Key                        | What                                                                  |
| -------------------------- | --------------------------------------------------------------------- |
| `TF_AWS_ACCESS_KEY_ID`     | AWS key Terraform uses for the SES identity, DKIM and MAIL FROM       |
| `TF_AWS_SECRET_ACCESS_KEY` | its secret                                                            |
| `SES_ACCESS_KEY_ID`        | key of an IAM user limited to `ses:SendRawEmail` (the app's mail key) |
| `SES_SECRET_ACCESS_KEY`    | its secret                                                            |
| `REGISTRY_CREDENTIALS`     | `<github user>:<token with read:packages>` for the image on GHCR      |
| `LICENSE_KEY`              | the EE license key; leave unset for the open-source core              |

### Other manual steps Terraform cannot do

- [ ] **SES production access request** in the SES console (Account dashboard) for `ses_region`
      (default `eu-central-1`). SES starts in the sandbox: it delivers only to verified addresses.
- [ ] **Registrar nameservers**: after the first apply, set `ns1`, `ns2`, `ns3.digitalocean.com` at the
      registrar (section 5). Until then the zone does not resolve and certificates are not issued.

State holds the sensitive variables, which is why the bucket stays private.

## 3. Variables and plan review

Plain values: copy `production.tfvars.example` to `production.tfvars` (gitignored). Variables:
`image_registry`, `image_repository`, `image_tag`; optional `otel_service_name`, `domain`
(`getsphericon.com`), `app_host_label` (`app`; `APP_URL` is `https://<app_host_label>.<domain>`),
`api_host_label` (`api`), `tracker_host` (empty means `t.<domain>`), `region`, `name`, `ses_region`,
`mail_from_label` (`mail`), `system_email_from` (default `noreply@<domain>`) and `dmarc_rua` (empty
omits `rua`). Secret variables come from the container (`TF_VAR_*` are set by the wrapper).

Plan and read it before applying:

```sh
mise run infra -- plan -out=tfplan
```

Check: only the expected resources (cluster, firewall, app, zone and records, SES identity); secrets
show as `(sensitive value)`; nothing is destroyed on a first run.

## 4. Apply

```sh
mise run infra -- apply tfplan
mise run infra -- output default_url
```

The first apply takes several minutes (cluster, then app, then the firewall rule that references
the app id). The deployment log shows the `migrate` pre-deploy job (`sphericon migrate`, river's
schema included) running before the service starts. The database accepts connections from the app
only.

## 5. Registrar nameservers (manual, once per zone)

```sh
doctl compute domain get getsphericon.com   # or: dig NS getsphericon.com @1.1.1.1
```

Set the three `ns*.digitalocean.com` names at the registrar. Propagation can take hours. Delegating
the whole zone also moves the apex to DigitalOcean DNS: the apex is reserved for the marketing
site (hosted elsewhere), the app creates no apex A or CNAME record, and the apex keeps only the
Google Workspace records, the SES identity records and DMARC. Add a marketing apex record in
`dns.tf` or at that host, never through the app.

## 6. Smoke test

Run after propagation and an active deployment. `<default_url>` is the Terraform output; replace the
domain if `domain` was overridden.

- [ ] Liveness: `curl -s <default_url>/healthz` returns `200`.
- [ ] Readiness on the platform URL: `curl -s <default_url>/readyz` returns `200` (database reachable).
- [ ] App host: `curl -sI https://app.getsphericon.com/` returns `200` with a platform certificate
      (the SPA); `curl -s https://app.getsphericon.com/readyz` returns `200`.
- [ ] Apex is not served by the app: `dig +short A getsphericon.com` returns no App Platform address.
- [ ] API host rewrite: `curl -si https://api.getsphericon.com/contacts` returns `401` with
      `content-type: application/problem+json` (the external API at `/api/contacts`, not the SPA).
- [ ] Tracker host: `curl -sI https://t.getsphericon.com/t.js` returns `200` with a JavaScript content type.
- [ ] Collect: `curl -si -X POST https://t.getsphericon.com/collect/<path>` without a key returns `401`
      problem+json; with a valid `x-collect-key` the event is accepted. `curl -si https://t.getsphericon.com/`
      is not the SPA (no rule for other paths).
- [ ] Email: trigger a password reset in the app (in the SES sandbox, to a verified address). The
      message arrives and its original headers show `dkim=pass`, `spf=pass` and `dmarc=pass`.
      The SES console (`ses_region`) shows identity Verified, DKIM Successful, MAIL FROM Success.
- [ ] Google records resolve: `dig +short MX getsphericon.com`, `dig +short TXT getsphericon.com`
      (exactly one SPF), `dig +short TXT google._domainkey.getsphericon.com` (complete value), and
      `dig +short MX mail.getsphericon.com`, `dig +short TXT _dmarc.getsphericon.com`.

## 7. Destroy and recreate

The environment is a test deployment: no deletion protection, no standby node.

```sh
mise run infra -- destroy
```

Destroy removes the app, the cluster **and all its data**, the zone, the SES identity and the DNS
records. The state bucket and its key stay (never destroy them with the environment). To recreate,
run sections 3, 4, 5 and 6 again, with these effects:

- **`ENCRYPTION_KEY`**: stays in the secret container and is reused as is. The database is new, so nothing stored is lost; but
  a different key against a restored database would make stored provider credentials unreadable.
- **Nameservers**: the zone is created again, and DigitalOcean may assign different nameservers.
  Compare `doctl compute domain get` with the registrar and update it if they differ; the app and
  certificates do not work until they match.
- **SES**: the identity is created again with new DKIM tokens, so verification repeats after the
  nameservers propagate. Production access is per AWS account and region and is kept.
- **Secrets**: all of them stay in the secret container and are reused as they are.

**Pending, operator only:** the environment has not yet been destroyed and recreated once by
following only this runbook. That acceptance check, and a first live pass of the smoke test (platform
URL, hostnames and rewrite, certificates, a delivered password-reset email passing DKIM and SPF,
Google records, secrets redacted in `plan`), need the operator's accounts and registrar and are
not covered by `mise run check:infra`.

## 8. Troubleshooting

- **API host returns the SPA or a 404 (host-match and rewrite).** Routing is by authority and path
  (`match.authority.exact`) plus `component.rewrite` to the `/api` prefix. `200` with HTML means the
  rewrite did not apply (the host fell through to `app.<domain>`); `404` with problem+json means the
  path joined wrongly (`/api` + `contacts` versus `/api/contacts`). The schema cannot show how the
  platform joins a `/` prefix, so only the smoke test confirms it; adjust the rule in `app.tf`.
  The app host also reaches `/api/*` (the binary is path-based); that is accepted.
- **Tracker host serves the SPA for `/`.** It must not: only `/t.js` and `/collect` have rules.
- **Google DKIM TXT (400 characters) missing or truncated.** DNS strings hold 255 characters; if
  DigitalOcean rejects the value or `dig` shows it cut, split it into 255-character quoted strings in
  `dns.tf`. The SES DKIM records are short CNAMEs and unaffected. Keep one apex SPF (Google); SES
  SPF belongs on `mail.<domain>`.
- **Certificates or hosts stay pending.** The nameservers are not delegated yet (section 5).
- **Connection budget.** A 1 GB Postgres allows 22 backend connections; each process opens two
  pools, `DB_MAX_OPEN_CONNS` (database/sql: ent, event bus, job workers' queries) + `PGX_MAX_CONNS`
  (river's own queries: fetch, completion, LISTEN, leader election) = 5 + 5 = 10 (the app defaults
  of 15 + 25 would not fit). river's 20 workers are fixed in code; they hold no pgx connection, so
  the pgx pool need not match them (the env table in `docs/self-hosting.md` says so); the cost is
  queued completions and 20 workers sharing 5 sql connections. The migrate job runs before the new
  instance and overlaps only the old one: it opens one uncapped `sql.Open` pool (goose, one
  statement at a time, 1-2 connections) and a river pool for the river migrator (also 1-2). So the
  worst case is old + new service = 20 of 22, leaving 2 for administration. Before a second
  service instance or any extra process, lower both pools (for example 3 + 3). No
  transaction-mode pooler: river needs LISTEN/NOTIFY. Symptom of overrun: `too many connections`
  in the deployment log.
- **Migrate job fails with "executable not found" or runs the server.** The job sets
  `run_command = "sphericon migrate"`. DigitalOcean documents that for Dockerfile builds a run
  command overrides the Dockerfile's entrypoint
  ([source](https://docs.digitalocean.com/products/app-platform/how-to/deploy-from-container-images/)),
  and the image has `ENTRYPOINT ["sphericon"]` with the binary on PATH (`/usr/local/bin`) and no
  CMD, so the full command is `sphericon migrate`; the service sets no run command and runs
  `sphericon` (the server). The docs say nothing about CMD, so this is to be confirmed in the first
  live deploy: the job log must show "migrations applied". If the platform instead appended the
  command to the entrypoint, use `run_command = "migrate"`.
- **Email not delivered.** The SES sandbox accepts only verified recipients until production access
  is granted; check identity status first. DigitalOcean blocks outbound SMTP, so the app uses the SES
  API (`SYSTEM_EMAIL_PROVIDER=ses`, see the env table in `docs/self-hosting.md`).
- **State lock or backend errors.** Spaces may not lock state; run one operator at a time.
