# Managed Postgres: one node, 1 GB, no standby (ADR 0028). Used directly, without a
# transaction-mode pooler, because river relies on LISTEN/NOTIFY.
resource "digitalocean_database_cluster" "pg" {
  name       = "${var.name}-pg"
  engine     = "pg"
  version    = "18" # the newest DigitalOcean offers; the repo's tests and dev stack run Postgres 18
  size       = "db-s-1vcpu-1gb"
  region     = var.region
  node_count = 1
}

# Only the app may connect: the single rule is of type "app", so nothing on the internet can.
resource "digitalocean_database_firewall" "pg" {
  cluster_id = digitalocean_database_cluster.pg.id

  rule {
    type  = "app"
    value = digitalocean_app.this.id
  }
}
