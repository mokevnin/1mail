# Sphericon hosted deployment: runbook

The hosted Sphericon deployment (DigitalOcean App Platform, Managed Postgres, DO DNS, AWS SES) is
described only by the Terraform in this directory. Why and what: [ADR 0028](../docs/adr/0028-saas-hosting-terraform-on-digitalocean-app-platform.md).
Follow the sections in order: prerequisites, manual steps, bootstrap, plan, apply, smoke test;
destroy and recreate and troubleshooting come after.

**DigitalOcean MCP servers (`.mcp.json`: apps, databases, droplets) are for read-only inspection
(status, logs, listings), never for changes.** Every change goes through Terraform, otherwise the
state drifts and the next apply reverts it.

## 1. Prerequisites

- `terraform`, `doctl` and the AWS CLI (`aws`) installed; `docker` for generating the encryption key.
- A DigitalOcean account, an AWS account (SES only), a GitHub account that can read the image on
  GHCR, and the registrar of `getsphericon.com`.
- `mise run check:infra` (fmt, `init -backend=false`, `validate`) needs no credentials; CI and the
  git hook run it.

## 2. Manual steps Terraform cannot do

One checklist. Nothing here is stored in git; secrets go to a password manager and the shell
environment only (`*.tfstate*`, `*.tfvars`, `backend.hcl` are gitignored).

- [ ] **DigitalOcean API token** (read and write): the provider reads `DIGITALOCEAN_TOKEN`.
- [ ] **Spaces state bucket** and its keys: section 3 (Terraform cannot create its own backend).
- [ ] **`ENCRYPTION_KEY`**: generate once, keep forever (see below).
- [ ] **GHCR credentials**: `<github user>:<token with read:packages>` as `TF_VAR_registry_credentials`.
- [ ] **AWS SES IAM user**: limited to `ses:SendRawEmail`, one access key, not Terraform's own key
      (`TF_VAR_ses_access_key_id` / `TF_VAR_ses_secret_access_key`). Terraform itself uses separate
      AWS credentials (`AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`) with SES and identity permissions.
- [ ] **SES production access request** in the SES console (Account dashboard) for `ses_region`
      (default `eu-central-1`). SES starts in the sandbox: it delivers only to verified addresses.
- [ ] **Registrar nameservers**: after the first apply, set `ns1`, `ns2`, `ns3.digitalocean.com` at the
      registrar (section 6). Until then the zone does not resolve and certificates are not issued.
- [ ] **`JWT_SECRET`, `BOOTSTRAP_TOKEN`**: `openssl rand -hex 32` each, kept in the password manager
      so a recreate can reuse them (a new `JWT_SECRET` only signs everyone out).
- [ ] **`LICENSE_KEY`**: the EE license key, or empty for the open-source core.

`ENCRYPTION_KEY` protects stored provider credentials: a new key makes them unreadable. Generate it
once, outside Terraform, and supply the same value on every apply and every recreate:

```sh
docker run --rm ghcr.io/<owner>/<image>:<tag> genkey   # or: go run ./cmd/server genkey
```

## 3. Bootstrap: the state bucket (once)

State lives in a private Spaces bucket (fra1). `doctl` cannot create buckets, so use the S3 API.

```sh
# 1. a full-access key (the secret is shown once), stored in an AWS CLI profile named "spaces"
#    (prompts for the key and secret; region and format can stay empty). Nothing goes into AWS_*
#    variables, which the aws provider would read for SES, or into shell history.
doctl spaces keys create sphericon-bootstrap --grants 'bucket=;permission=fullaccess'
aws configure --profile spaces

# 2. a private bucket (names are global across Spaces: pick a unique one)
aws --profile spaces s3api create-bucket --bucket sphericon-tfstate --acl private \
  --endpoint-url https://fra1.digitaloceanspaces.com

# 3. verify: the ACL has no AllUsers grant
aws --profile spaces s3api get-bucket-acl --bucket sphericon-tfstate --endpoint-url https://fra1.digitaloceanspaces.com

# 4. narrow the access: a key for this bucket only, then drop the bootstrap key
doctl spaces keys create sphericon-tfstate --grants 'bucket=sphericon-tfstate;permission=readwrite'
doctl spaces keys delete sphericon-bootstrap
```

Put the bucket-only key in `backend.hcl` (copy `backend.hcl.example`; gitignored, `chmod 600`),
never in `AWS_*` variables (the aws provider reads those for SES) and never in `-backend-config`
arguments (they land in shell history and process lists). Then initialise:

```sh
export DIGITALOCEAN_TOKEN=<API token>
export AWS_ACCESS_KEY_ID=<AWS key for Terraform's SES resources> AWS_SECRET_ACCESS_KEY=<its secret>
terraform -chdir=infra init -backend-config=backend.hcl
```

(With `-chdir=infra` the path resolves inside `infra/`, where `backend.hcl` lives.) Afterwards
delete the CLI profile: `aws configure set aws_access_key_id '' --profile spaces`, or edit
`~/.aws/credentials`.

State holds the sensitive variables, which is why the bucket stays private.

## 4. Variables and plan review

Plain values: copy `production.tfvars.example` to `production.tfvars`. Variables: `image_registry`,
`image_repository`, `image_tag`; optional `otel_service_name`, `domain` (`getsphericon.com`),
`app_host_label` (`app`; `APP_URL` is `https://<app_host_label>.<domain>`), `api_host_label` (`api`),
`tracker_host` (empty means `t.<domain>`), `region`, `name`, `ses_region`, `mail_from_label`
(`mail`), `system_email_from` (default `noreply@<domain>`) and `dmarc_rua` (empty omits `rua`).

Secrets, through the environment only:

```sh
export TF_VAR_registry_credentials='<github user>:<token with read:packages>'
export TF_VAR_jwt_secret='<from the password manager>'
export TF_VAR_bootstrap_token='<from the password manager>'
export TF_VAR_license_key='<license key, or empty>'
export TF_VAR_encryption_key='<the one generated key>'
export TF_VAR_ses_access_key_id='<SES send-only IAM user key>'
export TF_VAR_ses_secret_access_key='<its secret>'
```

Plan and read it before applying:

```sh
terraform -chdir=infra plan -var-file=production.tfvars -out=tfplan
```

Check: only the expected resources (cluster, firewall, app, zone and records, SES identity); secrets
show as `(sensitive value)`; nothing is destroyed on a first run.

## 5. Apply

```sh
terraform -chdir=infra apply tfplan
terraform -chdir=infra output default_url
```

The first apply takes several minutes (cluster, then app, then the firewall rule that references
the app id). The deployment log shows the `migrate` pre-deploy job (`sphericon migrate`, river's
schema included) running before the service starts. The database accepts connections from the app
only.

## 6. Registrar nameservers (manual, once per zone)

```sh
doctl compute domain get getsphericon.com   # or: dig NS getsphericon.com @1.1.1.1
```

Set the three `ns*.digitalocean.com` names at the registrar. Propagation can take hours. Delegating
the whole zone also moves the apex to DigitalOcean DNS: the apex is reserved for the marketing
site (hosted elsewhere), the app creates no apex A or CNAME record, and the apex keeps only the
Google Workspace records, the SES identity records and DMARC. Add a marketing apex record in
`dns.tf` or at that host, never through the app.

## 7. Smoke test

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

## 8. Destroy and recreate

The environment is a test deployment: no deletion protection, no standby node.

```sh
terraform -chdir=infra destroy -var-file=production.tfvars
```

Destroy removes the app, the cluster **and all its data**, the zone, the SES identity and the DNS
records. The state bucket and its key stay (never destroy them with the environment). To recreate,
run sections 4, 5, 6 and 7 again, with these effects:

- **`ENCRYPTION_KEY`**: supply the same value. The database is new, so nothing stored is lost; but
  a different key against a restored database would make stored provider credentials unreadable.
- **Nameservers**: the zone is created again, and DigitalOcean may assign different nameservers.
  Compare `doctl compute domain get` with the registrar and update it if they differ; the app and
  certificates do not work until they match.
- **SES**: the identity is created again with new DKIM tokens, so verification repeats after the
  nameservers propagate. Production access is per AWS account and region and is kept.
- **Secrets**: reuse `JWT_SECRET`, `BOOTSTRAP_TOKEN` and the SES and GHCR credentials from the
  password manager.

**Pending, operator only:** the environment has not yet been destroyed and recreated once by
following only this runbook. That acceptance check, and a first live pass of the smoke test (platform
URL, hostnames and rewrite, certificates, a delivered password-reset email passing DKIM and SPF,
Google records, secrets redacted in `plan`), need the operator's accounts and registrar and are
not covered by `mise run check:infra`.

## 9. Troubleshooting

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
- **Certificates or hosts stay pending.** The nameservers are not delegated yet (section 6).
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
