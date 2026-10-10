# Sphericon infrastructure (Terraform)

The hosted Sphericon deployment is described only here (ADR 0028). Nothing is created by hand in
the DigitalOcean console or through an MCP server, except the one-time state bucket below.

## Checks

`mise run check:infra` runs `terraform fmt -check`, `init -backend=false` and `validate`. It needs no
credentials and no network state; CI and the git hook run the same task.

## One-time bootstrap: the state bucket

State lives in a private DigitalOcean Spaces bucket, which Terraform cannot create for itself.
`doctl` has no bucket-create command, so use the S3 API (or the console).

1. Create a full-access Spaces key (the secret is shown once):

   ```sh
   doctl spaces keys create sphericon-bootstrap --grants 'bucket=;permission=fullaccess'
   ```

2. Map it to the AWS variable names the S3 tooling and Terraform backend read:

   ```sh
   export AWS_ACCESS_KEY_ID=<SPACES access key>
   export AWS_SECRET_ACCESS_KEY=<SPACES secret key>
   ```

3. Create the bucket, private (bucket names are global across Spaces, so pick a unique one):

   ```sh
   aws s3api create-bucket --bucket sphericon-tfstate --acl private \
     --endpoint-url https://fra1.digitaloceanspaces.com
   ```

4. Verify nothing is public: the ACL must have no `AllUsers` grant.

   ```sh
   aws s3api get-bucket-acl --bucket sphericon-tfstate \
     --endpoint-url https://fra1.digitaloceanspaces.com
   ```

5. Narrow the access: create a key scoped to that bucket only and delete the bootstrap key.

   ```sh
   doctl spaces keys create sphericon-tfstate --grants 'bucket=sphericon-tfstate;permission=readwrite'
   doctl spaces keys delete sphericon-bootstrap
   ```

   Keep this Spaces key apart from the AWS credentials of the SES provider: the `aws` provider
   reads `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`, so the state backend gets the Spaces key
   through `-backend-config` (see "Working with the module"). Export it as `SPACES_ACCESS_KEY_ID` /
   `SPACES_SECRET_ACCESS_KEY`.

## Working with the module

The provider token comes from the environment, never from a file:

```sh
export DIGITALOCEAN_TOKEN=<API token>
export AWS_ACCESS_KEY_ID=<AWS key with SES and identity permissions>   # the aws provider (SES)
export AWS_SECRET_ACCESS_KEY=<its secret>
terraform -chdir=infra init \
  -backend-config="bucket=sphericon-tfstate" \
  -backend-config="key=production/terraform.tfstate" \
  -backend-config="access_key=$SPACES_ACCESS_KEY_ID" \
  -backend-config="secret_key=$SPACES_SECRET_ACCESS_KEY"
```

`SPACES_*` hold the state-bucket key from the bootstrap; `AWS_*` are the AWS credentials for SES
only. Both stay in the environment.

Or copy `backend.hcl.example` to `backend.hcl` (gitignored) and pass `-backend-config=backend.hcl`.
State holds sensitive variables, so the bucket stays private; `*.tfstate*`, `*.tfvars` and
`backend.hcl` are gitignored.

## Application and database

`database.tf` creates the Managed Postgres cluster (PostgreSQL 18, `db-s-1vcpu-1gb`, one node, fra1)
and a firewall whose only rule is the app. `app.tf` creates the App Platform app: one service
(`web`, shared CPU 1 vCPU / 1 GB, readiness on `/readyz`) and one `PRE_DEPLOY` job (`migrate`) that
runs `sphericon migrate` from the same image before each new service instance starts. The database
is attached through the app spec, so `DATABASE_URL` is injected as `${db.DATABASE_URL}`.
`AUTO_MIGRATE` stays unset and there is no worker component: river and watermill run in the service.

### Variables

Plain values go in `production.tfvars` (copy `production.tfvars.example`; gitignored):
`image_registry` (GHCR owner), `image_repository`, `image_tag`, and optionally `otel_service_name`,
`domain` (defaults to `getsphericon.com`; `APP_URL` is `https://<domain>`), `api_host_label`
(`api`), `tracker_host` (empty means `t.<domain>`), `region` and `name`, and for email `ses_region` (default `eu-central-1`), `mail_from_label`
(`mail`), `system_email_from` (default `noreply@getsphericon.com`, on the apex) and `dmarc_rua`
(report mailbox, empty omits `rua`).

Secrets are sensitive variables, passed through the environment so they never touch a file:

```sh
export TF_VAR_registry_credentials='<github user>:<token with read:packages>'
export TF_VAR_jwt_secret="$(openssl rand -hex 32)"
export TF_VAR_bootstrap_token="$(openssl rand -hex 32)"
export TF_VAR_license_key='<license key, or empty for the open-source core>'
export TF_VAR_ses_access_key_id='<access key of the SES send-only IAM user>'
export TF_VAR_ses_secret_access_key='<its secret>'
export TF_VAR_encryption_key='<see below>'
```

`ENCRYPTION_KEY` is generated **once**, outside Terraform, and kept in your password manager. It
protects stored provider credentials: a new key makes them unreadable, so never regenerate it for
an existing environment (the same value is supplied again on every re-apply):

```sh
docker run --rm ghcr.io/<owner>/<image>:<tag> genkey   # or: go run ./cmd/genkey
```

All secrets become `SECRET` env values in the app spec (encrypted by the platform) and are
redacted in `plan` output, but they are stored in state, which is why the bucket is private.

### Plan and apply

```sh
terraform -chdir=infra plan  -var-file=production.tfvars -out=tfplan
terraform -chdir=infra apply tfplan
terraform -chdir=infra output default_url
```

Read the plan before applying. The first apply takes several minutes (cluster, then app, then the
firewall rule that references the app id). Smoke test: `curl <default_url>/healthz` and `/readyz`;
the deployment log of the app shows the `migrate` job running first.

### Connection budget

A 1 GB Postgres plan allows 22 backend connections. Every process opens two pools, so one process
uses at most `DB_MAX_OPEN_CONNS + PGX_MAX_CONNS` = 5 + 5 = 10 (the app defaults of 15 + 25 would
not fit). Peaks: the old service plus the migrate job (10 + 10 at the pool caps, really about 2
for the job), then the old plus the new service during a rolling deploy (10 + 10). The new service
starts only after the job exits, so at most two full processes run at once: 20 of 22, leaving 2
for administration. Three concurrent processes (3 x 10 = 30) do not fit: before adding a second
service instance, lower both pools (for example 3 + 3). river's 20 workers share a 5-connection
pool, so jobs queue on the pool instead of running in parallel. No transaction-mode pooler is
used because river needs LISTEN/NOTIFY.

### Pending live verification

Needs the operator's account and is not covered by `mise run check:infra`: the default URL answers
liveness and readiness after apply, the pre-deploy job applied the migrations (including river's
schema), the database is unreachable from outside the app, secrets are redacted in plan output,
and destroy followed by re-apply recreates the environment.

## DNS and hostnames

`dns.tf` creates the `getsphericon.com` zone (`domain`) and the four Google Workspace mail records
(MX `@` `smtp.google.com.` priority 1, the apex SPF TXT, the `google-site-verification` TXT and the
`google._domainkey` DKIM TXT). The apex SPF is the only one: SES (ticket #204) authenticates
through its MAIL FROM subdomain and must not add a second apex SPF record.

`app.tf` attaches three domains to the app, each with `zone` set so the platform creates the DNS
records itself and issues the TLS certificate (no record or certificate is handled by hand), and an
ingress that routes by authority and path to the one service:

| Host              | Paths               | Service path                        |
| ----------------- | ------------------- | ----------------------------------- |
| `<domain>` (apex) | `/`                 | unchanged: SPA, `/site/*`           |
| `api.<domain>`    | `/`                 | rewritten to the `/api` prefix      |
| `t.<domain>`      | `/t.js`, `/collect` | unchanged; other paths have no rule |

The provider schema expresses the host match (`match.authority.exact`) together with the rewrite
(`component.rewrite`), so nothing is missing at the schema level. What the schema cannot show is
how the platform joins the rewrite to the trimmed path for a `/` prefix (`/api` + `x` versus
`/api/x`); that, and the tracker rules, are confirmed only by the smoke test below. The apex also
reaches `/api/*` (the binary is path-based); that is accepted. `APP_URL` is `https://<domain>`;
there are no new environment variables (the app reads only `APP_URL`).

### Pointing the registrar at DigitalOcean (manual, once)

After the first apply, read the zone's nameservers and set them at the registrar of
`getsphericon.com` (the zone stays unresolvable, and platform certificates are not issued, until
then):

```sh
doctl compute domain get getsphericon.com   # or: dig NS getsphericon.com @1.1.1.1
```

The values are `ns1.digitalocean.com`, `ns2.digitalocean.com` and `ns3.digitalocean.com`.
Propagation can take hours.

### Smoke test

Run after the nameservers have propagated and the deployment is active. Replace the domain if
`domain` was overridden.

- [ ] `curl -sI https://getsphericon.com/` returns `200` over HTTPS with a platform certificate
      (the SPA); `curl -s https://getsphericon.com/readyz` returns `200`.
- [ ] `curl -si https://api.getsphericon.com/contacts` returns `401` with
      `content-type: application/problem+json` (the external API, not the SPA's HTML). A `200`
      with HTML means the rewrite did not apply; a `404` with problem+json means it produced a wrong
      path (check `/api` + `/contacts` joining).
- [ ] `curl -sI https://t.getsphericon.com/t.js` returns `200` with a JavaScript content type.
- [ ] `curl -si -X POST https://t.getsphericon.com/collect/<path>` without a collect key returns
      `401` problem+json (reached collect); with a valid `x-collect-key` an event is accepted.
- [ ] `curl -si https://t.getsphericon.com/` is not the SPA (no rule for other paths).
- [ ] MX, SPF, `google-site-verification` and `google._domainkey` resolve:
      `dig +short MX getsphericon.com`, `dig +short TXT getsphericon.com`,
      `dig +short TXT google._domainkey.getsphericon.com`. Check that the 400-character DKIM
      value comes back complete (DNS splits long TXT strings; if DigitalOcean rejects it, split it
      into 255-character quoted strings).

### Pending live verification (DNS and hostnames)

Not covered by `mise run check:infra` and needs the operator's account and registrar: apex over
HTTPS, the api host rewrite verified live, the tracker host serving `/t.js` and accepting collect,
platform-issued certificates, the platform-created records, and the Google records resolving.

## System email (SES)

DigitalOcean blocks outbound SMTP, so the app sends platform mail (password reset, invitations)
through the SES HTTPS API: `SYSTEM_EMAIL_PROVIDER=ses`, `SYSTEM_EMAIL_FROM`, `SES_REGION` (plain)
and `SES_ACCESS_KEY_ID` / `SES_SECRET_ACCESS_KEY` (sensitive variables, `SECRET` env values) in
`app.tf`. All five are in the env table of `docs/self-hosting.md`.

`ses.tf` creates the SES domain identity for the apex, Easy DKIM and the custom MAIL FROM domain
`mail.<domain>` (AWS provider, credentials only from `AWS_*` in the environment), and in the
DigitalOcean zone: the `_amazonses` verification TXT, the three DKIM CNAMEs, the MAIL FROM MX
(`feedback-smtp.<ses_region>.amazonses.com`, priority 10), its SPF TXT (`include:amazonses.com`) and
the `_dmarc` TXT (`p=none`: DMARC only reports until the setup is proven). The apex SPF stays the
single Google Workspace record; SES SPF is on the MAIL FROM subdomain only. The DKIM records are
three short CNAMEs, so the 255-character TXT string limit does not apply to them (it only matters
for the Google DKIM TXT above).

One-time operator steps (outside Terraform):

1. Create an IAM user limited to SES sending (`ses:SendRawEmail`), make an access key, and pass it as
   `TF_VAR_ses_access_key_id` / `TF_VAR_ses_secret_access_key`. Do not reuse Terraform's own key.
2. SES starts in the sandbox: it delivers only to verified addresses and caps volume. Request
   production access in the SES console (Account dashboard) for `ses_region` before sending to
   real users.

### Verify

- [ ] SES console (region `ses_region`) shows the identity `getsphericon.com` as Verified, DKIM
      Successful and MAIL FROM Success (needs the zone delegated; propagation can take hours).
- [ ] `dig +short MX mail.getsphericon.com`, `dig +short TXT mail.getsphericon.com` and
      `dig +short TXT _dmarc.getsphericon.com` return the records; the apex still has exactly one
      SPF TXT.
- [ ] A password reset in the deployed app arrives (in the sandbox, to a verified address) and its
      original headers show `dkim=pass`, `spf=pass` and `dmarc=pass`.
- [ ] The SES keys do not appear in plain text in `terraform plan` output.

### Pending live verification (SES)

Needs the operator's AWS and DigitalOcean accounts: SES reports the domain verified with DKIM
passing, a deployed password-reset email arrives and passes DKIM and SPF, and the production-access
request is granted. `mise run check:infra` only validates the configuration.
