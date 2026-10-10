output "app_id" {
  description = "App Platform app id."
  value       = digitalocean_app.this.id
}

output "default_url" {
  description = "Platform URL of the app (before any custom domain): smoke-test /healthz and /readyz here."
  value       = digitalocean_app.this.default_ingress
}

output "database_cluster_id" {
  description = "Managed Postgres cluster id."
  value       = digitalocean_database_cluster.pg.id
}
