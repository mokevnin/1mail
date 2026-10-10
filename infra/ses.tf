# System email through Amazon SES (ADR 0028). The identity is the apex domain, so mail is sent from
# an address such as noreply@<domain>. The records live in the DigitalOcean zone of dns.tf.

locals {
  mail_from_domain = "${var.mail_from_label}.${var.domain}"
}

resource "aws_ses_domain_identity" "this" {
  domain = var.domain
}

resource "aws_ses_domain_dkim" "this" {
  domain = aws_ses_domain_identity.this.domain
}

# Bounces and SPF alignment use the MAIL FROM subdomain (envelope sender), not the apex.
resource "aws_ses_domain_mail_from" "this" {
  domain           = aws_ses_domain_identity.this.domain
  mail_from_domain = local.mail_from_domain
}

resource "digitalocean_record" "ses_verification" {
  domain = digitalocean_domain.this.id
  type   = "TXT"
  name   = "_amazonses"
  value  = aws_ses_domain_identity.this.verification_token
  ttl    = local.dns_ttl
}

# Easy DKIM: three CNAMEs to <token>.dkim.amazonses.com.
resource "digitalocean_record" "ses_dkim" {
  count = 3

  domain = digitalocean_domain.this.id
  type   = "CNAME"
  name   = "${aws_ses_domain_dkim.this.dkim_tokens[count.index]}._domainkey"
  value  = "${aws_ses_domain_dkim.this.dkim_tokens[count.index]}.dkim.amazonses.com."
  ttl    = local.dns_ttl
}

resource "digitalocean_record" "ses_mail_from_mx" {
  domain   = digitalocean_domain.this.id
  type     = "MX"
  name     = var.mail_from_label
  value    = "feedback-smtp.${var.ses_region}.amazonses.com."
  priority = 10
  ttl      = local.dns_ttl
}

# SPF of the MAIL FROM subdomain only. The apex keeps the single Google Workspace SPF (dns.tf).
resource "digitalocean_record" "ses_mail_from_spf" {
  domain = digitalocean_domain.this.id
  type   = "TXT"
  name   = var.mail_from_label
  value  = "v=spf1 include:amazonses.com ~all"
  ttl    = local.dns_ttl
}

# DMARC passes through aligned DKIM (the apex identity signs the From domain) or aligned SPF
# (relaxed: the MAIL FROM subdomain shares the organizational domain).
resource "digitalocean_record" "dmarc" {
  domain = digitalocean_domain.this.id
  type   = "TXT"
  name   = "_dmarc"
  value  = var.dmarc_rua != "" ? "v=DMARC1; p=none; rua=mailto:${var.dmarc_rua}" : "v=DMARC1; p=none"
  ttl    = local.dns_ttl
}
