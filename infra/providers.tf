# The token is read from the DIGITALOCEAN_TOKEN environment variable (`mise run infra` exports
# it from `doctl auth token`).
provider "digitalocean" {}

# SES lives in AWS. The credentials are explicit variables, not AWS_ACCESS_KEY_ID /
# AWS_SECRET_ACCESS_KEY: the s3 state backend reads those for the Spaces key. `mise run infra`
# fills the variables from the Secrets Manager container; nothing is committed. Only SES
# resources use this provider.
provider "aws" {
  region     = var.ses_region
  access_key = var.aws_access_key_id
  secret_key = var.aws_secret_access_key
}
