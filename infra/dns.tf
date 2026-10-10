# The zone is the root of every hostname and mail record (ADR 0028). Until the registrar's
# nameservers point at DigitalOcean (a one-time manual step, see README.md) nothing here resolves.
locals {
  dns_ttl = 3600
}

resource "digitalocean_domain" "this" {
  name = var.domain
}

# Google Workspace mail records. The values are public.

resource "digitalocean_record" "google_mx" {
  domain   = digitalocean_domain.this.id
  type     = "MX"
  name     = "@"
  value    = "smtp.google.com."
  priority = 1
  ttl      = local.dns_ttl
}

# The one and only SPF record of the apex: a second apex SPF TXT makes both invalid (RFC 7208).
# SES authenticates through its own MAIL FROM subdomain, so it must NOT add an apex
# SPF record.
resource "digitalocean_record" "google_spf" {
  domain = digitalocean_domain.this.id
  type   = "TXT"
  name   = "@"
  value  = "v=spf1 include:_spf.google.com ~all"
  ttl    = local.dns_ttl
}

resource "digitalocean_record" "google_site_verification" {
  domain = digitalocean_domain.this.id
  type   = "TXT"
  name   = "@"
  value  = "google-site-verification=xrqY2VwkPiLSLIP08QxCtnAQJ4ZKwCQJfyanLRK2YFM"
  ttl    = local.dns_ttl
}

resource "digitalocean_record" "google_dkim" {
  domain = digitalocean_domain.this.id
  type   = "TXT"
  name   = "google._domainkey"
  value  = "v=DKIM1;k=rsa;p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxzYPVFpRhJvSY0r5fdPbbpJsm5oxGmKB5KeJSsKYzW1ukIR52voGVuCLEYNTxLGtTCSrlAlziDl+6Skn+nckkVAoGLtfSukGZnRpuS1vv6ZDKABvesIz9CePW9s92nZ/eTWO6pC6XKJQtoKY1QOsh49mAdIV5czWxzBF3J/bgasfz8IORnthiciOFtB8VqmSCN7rgHFcCuyjy0FynOzdNQurStOmR4u8vA/dZWPg8XnsIlMdm7OsdI3pIpedxw6J7peLTHUTgbZxfOwPWrIEAvAIDftGhE7P+rEFngVmg2UdhA6pkcMiAjWFYyRg6qXyTBTTzjynIZMiOiKTOQH6LQIDAQAB"
  ttl    = local.dns_ttl
}
