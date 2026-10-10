# The SaaS runs on DigitalOcean App Platform, provisioned only through Terraform

The hosted offering (Sphericon, `getsphericon.com`) is described entirely in Terraform under
`infra/`: nothing is created by hand in the DigitalOcean console or through an MCP server. One
root module, one environment (production; it is a test deployment that is destroyed and recreated
freely, so there is no deletion protection and no standby node). State lives in a private DO
Spaces bucket through the S3 backend; the bucket is the single manual bootstrap step (`doctl`),
because the backend cannot create its own storage.

The runtime is one App Platform **service** plus one `PRE_DEPLOY` **job** running `sphericon migrate`
(`AUTO_MIGRATE` stays unset). river and watermill run inside the server process, so there is no
separate worker component. The database is a DO Managed Postgres cluster (fra1, 1 GB, single
node) attached to the app and firewalled to it. DNS for `getsphericon.com` lives in DO DNS and is
managed by the same module, including the SES DKIM/SPF/DMARC records, because DO has no email
service and outbound SMTP is blocked: system mail goes through SES over HTTPS. The image is pulled
from GHCR; the image repository and tag and `OTEL_SERVICE_NAME` are Terraform variables, so the
infra does not depend on the rename of the product.

Hosts: `app.getsphericon.com` serves the SPA and `/site/*` (`APP_URL` is that origin); `api.getsphericon.com` reaches `/api/*` through an
ingress rule that matches the authority and rewrites the path (the edge rewrite described for
`api.sphericon.localhost`); the tracker (`/t.js`, `/collect/*`) gets its own hostname so it can move to
another edge later. The apex `getsphericon.com` is reserved for a marketing site hosted elsewhere:
the app creates no apex record, and the apex keeps only the Google Workspace mail records and the
SES identity and DMARC.

## Considered Options

- **DOKS + the existing Helm chart.** Rejected for now: a node pool and a load balancer cost
  roughly two to three times more at this size and add cert-manager and ingress to run. The chart
  stays for self-hosters; this decision is revisited when replica count or networking outgrows
  App Platform.
- **Droplets + Docker.** Rejected: manual deploys and scaling.
- **Terraform Cloud / Cloudflare R2 for state.** Rejected: Spaces keeps everything in one account.

## Consequences

- A 1 GB Postgres allows 22 backend connections. The app spec must lower `DB_MAX_OPEN_CONNS` and
  `PGX_MAX_CONNS` (the defaults add up to 40), and leave room for the pre-deploy job and for the
  old and new instance overlapping during a deploy. No PgBouncer in transaction mode: river
  relies on LISTEN/NOTIFY.
- Secrets (`JWT_SECRET`, `ENCRYPTION_KEY`, `LICENSE_KEY`, SES keys, `BOOTSTRAP_TOKEN`) enter as
  sensitive variables and therefore appear in state, which is why the bucket is private.
  `ENCRYPTION_KEY` is generated once outside Terraform: losing it makes stored secrets unreadable.
- Spaces may not support state locking; with a single operator that is accepted.
- **Open risk: customer CNAME tracking domains.** Letting customers point their own subdomain at
  the tracker needs per-tenant domains and certificates. App Platform's per-app domain limit and
  whether a domain can be added through the API without redeploying the spec are unconfirmed
  (only forum answers, which disagree). If they do not fit, that feature needs a different edge
  (for example Caddy with on-demand TLS) in front of the tracker hostname. It is a separate
  decision and does not block this one.
