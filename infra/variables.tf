variable "region" {
  description = "DigitalOcean region for the app and the database (they must share it for private networking)."
  type        = string
  default     = "fra1"
}

variable "name" {
  description = "Base name of the app and the database cluster."
  type        = string
  default     = "sphericon"
}

# Image (pulled from GHCR). Repository and tag are variables so infra does not wait on the rename.

variable "image_registry" {
  description = "GHCR owner (user or organisation) that holds the image, e.g. \"mokevnin\"."
  type        = string
}

variable "image_repository" {
  description = "Image name under the owner, e.g. \"sphericon\"."
  type        = string
}

variable "image_tag" {
  description = "Image tag to deploy. Pin a release tag; changing it redeploys the app."
  type        = string
}

variable "registry_credentials" {
  description = "GHCR pull credentials as \"<username>:<token>\" (a token with read:packages). Leave empty only if the package is public."
  type        = string
  default     = ""
  sensitive   = true
}

# Application settings.

variable "otel_service_name" {
  description = "OTEL_SERVICE_NAME reported to OpenTelemetry; follows the product name without an infra change."
  type        = string
  default     = "sphericon"
}

variable "domain" {
  description = "Apex domain of the deployment. The zone is created in DigitalOcean DNS; point the registrar's nameservers at it once (README.md). APP_URL is https://<domain>."
  type        = string
  default     = "getsphericon.com"
}

variable "api_host_label" {
  description = "Label of the external API host: <label>.<domain>, rewritten to the /api prefix of the service."
  type        = string
  default     = "api"
}

variable "tracker_host" {
  description = "Hostname that serves the tracker script (/t.js) and collect (/collect/*). Empty means t.<domain>."
  type        = string
  default     = ""
}

variable "app_port" {
  description = "PORT the server listens on; also the service http_port."
  type        = number
  default     = 3000
}

variable "db_max_open_conns" {
  description = "DB_MAX_OPEN_CONNS: the database/sql pool of one process. See the connection budget in app.tf."
  type        = number
  default     = 5
}

variable "pgx_max_conns" {
  description = "PGX_MAX_CONNS: the river pool of one process. See the connection budget in app.tf."
  type        = number
  default     = 5
}

# Secrets. All of them come from outside Terraform and end up as SECRET env values (encrypted by
# the platform). They are in state, which is why the state bucket is private.

variable "jwt_secret" {
  description = "JWT_SECRET: at least 32 characters (openssl rand -hex 32)."
  type        = string
  sensitive   = true
}

variable "encryption_key" {
  description = "ENCRYPTION_KEY: base64 Tink keyset, generated ONCE outside Terraform (sphericon genkey). Losing or replacing it makes stored provider credentials unreadable."
  type        = string
  sensitive   = true
}

variable "license_key" {
  description = "LICENSE_KEY: Enterprise Edition license key. Empty runs the open-source core."
  type        = string
  default     = ""
  sensitive   = true
}

variable "bootstrap_token" {
  description = "BOOTSTRAP_TOKEN: external-API bootstrap token."
  type        = string
  sensitive   = true
}

# System email through SES (ADR 0028): DigitalOcean blocks outbound SMTP, so the app sends over
# the SES HTTPS API.

variable "ses_region" {
  description = "AWS region of the SES identity and of the app's SES_REGION (e.g. eu-central-1). Terraform's own AWS credentials come from the AWS_* environment variables."
  type        = string
  default     = "eu-central-1"
}

variable "mail_from_label" {
  description = "Label of the SES MAIL FROM subdomain: <label>.<domain>. SPF for SES lives here, never on the apex (the apex SPF belongs to Google Workspace)."
  type        = string
  default     = "mail"
}

variable "system_email_from" {
  description = "SYSTEM_EMAIL_FROM: sender address of platform mail (password reset, invitations). Must be on the apex domain, which is the verified SES identity."
  type        = string
  default     = "noreply@getsphericon.com"
}

variable "dmarc_rua" {
  description = "Mailbox that receives DMARC aggregate reports (rua). Empty omits the tag."
  type        = string
  default     = ""
}

variable "ses_access_key_id" {
  description = "SES_ACCESS_KEY_ID: access key of an IAM user limited to SES sending. Not the key Terraform itself uses."
  type        = string
  sensitive   = true
}

variable "ses_secret_access_key" {
  description = "SES_SECRET_ACCESS_KEY: secret of the IAM user above."
  type        = string
  sensitive   = true
}
