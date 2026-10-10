terraform {
  required_version = ">= 1.16.0"

  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }

  # State lives in a private DigitalOcean Spaces bucket (ADR 0028). Spaces speaks the S3 API, so
  # the s3 backend is used with every AWS-only check skipped. Bucket and key are passed by
  # `mise run infra` (non-secret -backend-config values); the Spaces key reaches the backend as
  # AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY from the doctl Secrets Manager container, never from
  # a file or an argument. The aws provider therefore takes its own credentials through
  # variables (providers.tf).
  backend "s3" {
    endpoints = {
      s3 = "https://fra1.digitaloceanspaces.com"
    }
    region = "us-east-1" # required by the backend, ignored by Spaces

    skip_credentials_validation = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    skip_s3_checksum            = true
    use_path_style              = true
  }
}
