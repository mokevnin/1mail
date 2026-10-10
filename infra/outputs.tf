output "default_url" {
  description = "Platform URL of the app (before any custom domain): smoke-test /healthz and /readyz here."
  value       = digitalocean_app.this.default_ingress
}

output "app_url" {
  description = "Public URL of the web app (APP_URL). The apex domain is not served by the app."
  value       = "https://${local.app_host}"
}
