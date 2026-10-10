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

variable "app_url" {
  description = "APP_URL: the public origin used in auth cookies and in links in emails. Point it at the apex once DNS is attached."
  type        = string
  default     = "https://getsphericon.com"
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
