# The Route 53 zone is the root of every hostname and mail record (ADR 0029). Until the registrar's
# nameservers point at it (a one-time manual step, see README.md) nothing here resolves and the
# certificate is not validated.
locals {
  dns_ttl = 3600

  app_host     = "${var.app_host_label}.${var.domain}"
  api_host     = "${var.api_host_label}.${var.domain}"
  tracker_host = "${var.tracker_host_label}.${var.domain}"

  # A TXT string holds at most 255 characters: longer values are stored as several quoted strings.
  google_dkim = "v=DKIM1;k=rsa;p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxzYPVFpRhJvSY0r5fdPbbpJsm5oxGmKB5KeJSsKYzW1ukIR52voGVuCLEYNTxLGtTCSrlAlziDl+6Skn+nckkVAoGLtfSukGZnRpuS1vv6ZDKABvesIz9CePW9s92nZ/eTWO6pC6XKJQtoKY1QOsh49mAdIV5czWxzBF3J/bgasfz8IORnthiciOFtB8VqmSCN7rgHFcCuyjy0FynOzdNQurStOmR4u8vA/dZWPg8XnsIlMdm7OsdI3pIpedxw6J7peLTHUTgbZxfOwPWrIEAvAIDftGhE7P+rEFngVmg2UdhA6pkcMiAjWFYyRg6qXyTBTTzjynIZMiOiKTOQH6LQIDAQAB"
}

resource "aws_route53_zone" "this" {
  name = var.domain
}

# Google Workspace mail records. The values are public.

resource "aws_route53_record" "google_mx" {
  zone_id = aws_route53_zone.this.zone_id
  name    = var.domain
  type    = "MX"
  ttl     = local.dns_ttl
  records = ["1 smtp.google.com."]
}

# The apex TXT record set: the one and only SPF record (a second apex SPF TXT makes both invalid,
# RFC 7208) and the Google site verification. SES authenticates through its own MAIL FROM
# subdomain, so it must NOT add an apex SPF record.
resource "aws_route53_record" "apex_txt" {
  zone_id = aws_route53_zone.this.zone_id
  name    = var.domain
  type    = "TXT"
  ttl     = local.dns_ttl
  records = [
    "v=spf1 include:_spf.google.com ~all",
    "google-site-verification=xrqY2VwkPiLSLIP08QxCtnAQJ4ZKwCQJfyanLRK2YFM",
  ]
}

resource "aws_route53_record" "google_dkim" {
  zone_id = aws_route53_zone.this.zone_id
  name    = "google._domainkey.${var.domain}"
  type    = "TXT"
  ttl     = local.dns_ttl
  records = [join(" ", [for chunk in regexall(".{1,255}", local.google_dkim) : "\"${chunk}\""])]
}

# The app, API and tracker hosts all point at the load balancer; it routes by Host (alb.tf).
# The apex has no record here on purpose: it is reserved for the marketing site.
resource "aws_route53_record" "hosts" {
  for_each = toset([local.app_host, local.api_host, local.tracker_host])

  zone_id = aws_route53_zone.this.zone_id
  name    = each.key
  type    = "A"

  alias {
    name                   = aws_lb.this.dns_name
    zone_id                = aws_lb.this.zone_id
    evaluate_target_health = true
  }
}
