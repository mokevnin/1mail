# The token is read from the DIGITALOCEAN_TOKEN environment variable.
provider "digitalocean" {}

# SES lives in AWS. Credentials come only from the environment (AWS_ACCESS_KEY_ID,
# AWS_SECRET_ACCESS_KEY); nothing is committed. Only SES resources use this provider.
provider "aws" {
  region = var.ses_region
}
