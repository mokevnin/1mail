locals {
  # Connection budget. A 1 GB Postgres plan allows 22 backend connections, and every process opens
  # two pools: database/sql (DB_MAX_OPEN_CONNS) and river's pgx pool (PGX_MAX_CONNS), so one
  # process peaks at db_max_open_conns + pgx_max_conns (5 + 5 = 10).
  #
  #   steady state       service                          10
  #   pre-deploy job     service (old) + migrate job      10 + 10 at the pool caps
  #                      (really 1-2 each: goose and river's migrator run one statement at a time)
  #   rolling deploy     old + new service                10 + 10
  #   worst case         two processes                    20 of 22, leaving 2 for admin/monitoring
  #
  # The new service starts only after the PRE_DEPLOY job has exited, so the job never overlaps the
  # new instance: at most two full processes exist at once. Three (3 x 10 = 30) would NOT fit, so
  # a second service instance requires smaller pools first. river runs 20 workers (5 + 10 + 5) on
  # a 5-connection pool: workers wait for a connection instead of running in parallel, which is
  # acceptable for a test production.
  secret_env = {
    JWT_SECRET      = var.jwt_secret
    ENCRYPTION_KEY  = var.encryption_key
    LICENSE_KEY     = var.license_key
    BOOTSTRAP_TOKEN = var.bootstrap_token

    SES_ACCESS_KEY_ID     = var.ses_access_key_id
    SES_SECRET_ACCESS_KEY = var.ses_secret_access_key
  }

  plain_env = {
    APP_URL           = "https://${var.domain}"
    PORT              = tostring(var.app_port)
    OTEL_SERVICE_NAME = var.otel_service_name
    DB_MAX_OPEN_CONNS = tostring(var.db_max_open_conns)
    PGX_MAX_CONNS     = tostring(var.pgx_max_conns)

    SYSTEM_EMAIL_PROVIDER = "ses"
    SYSTEM_EMAIL_FROM     = var.system_email_from
    SES_REGION            = var.ses_region
  }

  registry_credentials = var.registry_credentials != "" ? var.registry_credentials : null

  api_host     = "${var.api_host_label}.${var.domain}"
  tracker_host = var.tracker_host != "" ? var.tracker_host : "t.${var.domain}"
}

resource "digitalocean_app" "this" {
  spec {
    name   = var.name
    region = var.region

    # Lets the app reach the cluster and injects its connection string as ${db.DATABASE_URL}
    # (sslmode=require). AUTO_MIGRATE stays unset: the pre-deploy job migrates once.
    database {
      name         = "db"
      engine       = "PG"
      production   = true
      cluster_name = digitalocean_database_cluster.pg.name
    }

    # App-level env reaches the service and the job alike, so both read the same database and pool
    # settings (the migration command loads the full production config).
    env {
      key   = "DATABASE_URL"
      value = "$${db.DATABASE_URL}"
      scope = "RUN_TIME"
      type  = "GENERAL"
    }

    dynamic "env" {
      for_each = local.plain_env
      content {
        key   = env.key
        value = env.value
        scope = "RUN_TIME"
        type  = "GENERAL"
      }
    }

    dynamic "env" {
      for_each = local.secret_env
      content {
        key   = env.key
        value = env.value
        scope = "RUN_TIME"
        type  = "SECRET"
      }
    }

    # Hostnames. `zone` makes the platform create and manage the DNS record in the zone of
    # dns.tf, and it issues a TLS certificate for each name. The apex is the primary domain.
    domain {
      name = var.domain
      type = "PRIMARY"
      zone = digitalocean_domain.this.name
    }

    domain {
      name = local.api_host
      type = "ALIAS"
      zone = digitalocean_domain.this.name
    }

    domain {
      name = local.tracker_host
      type = "ALIAS"
      zone = digitalocean_domain.this.name
    }

    # Ingress by authority and path (the binary stays path-based: /site, /api, /collect, /t.js).
    ingress {
      # api.<domain>/x -> service /api/x. The platform trims the matched prefix ("/") and puts
      # `rewrite` in its place; preserve_path_prefix must stay unset next to a rewrite. The exact
      # joining ("/api" + "x" vs "/api/x") is verified by the smoke test, not by the schema.
      rule {
        match {
          authority {
            exact = local.api_host
          }
          path {
            prefix = "/"
          }
        }
        component {
          name    = "web"
          rewrite = "/api"
        }
      }

      # The tracker host exposes only the script and the collect endpoint; other paths get no rule.
      rule {
        match {
          authority {
            exact = local.tracker_host
          }
          path {
            prefix = "/t.js"
          }
        }
        component {
          name                 = "web"
          preserve_path_prefix = true
        }
      }

      rule {
        match {
          authority {
            exact = local.tracker_host
          }
          path {
            prefix = "/collect"
          }
        }
        component {
          name                 = "web"
          preserve_path_prefix = true
        }
      }

      # The apex serves the SPA and /site/*.
      rule {
        match {
          authority {
            exact = var.domain
          }
          path {
            prefix = "/"
          }
        }
        component {
          name = "web"
        }
      }
    }

    # One service: HTTP, river and watermill all run in this process, so there is no worker.
    service {
      name               = "web"
      instance_size_slug = "apps-s-1vcpu-1gb"
      instance_count     = 1
      http_port          = var.app_port

      image {
        registry_type        = "GHCR"
        registry             = var.image_registry
        repository           = var.image_repository
        tag                  = var.image_tag
        registry_credentials = local.registry_credentials
      }

      # /readyz pings the database, so a deploy that cannot reach Postgres is not routed to.
      health_check {
        http_path             = "/readyz"
        initial_delay_seconds = 10
        period_seconds        = 10
        timeout_seconds       = 3
        failure_threshold     = 3
      }
    }

    # Applies the goose and river migrations once, before the new service instance starts.
    job {
      name               = "migrate"
      kind               = "PRE_DEPLOY"
      instance_size_slug = "apps-s-1vcpu-0.5gb"
      instance_count     = 1
      run_command        = "sphericon migrate"

      image {
        registry_type        = "GHCR"
        registry             = var.image_registry
        repository           = var.image_repository
        tag                  = var.image_tag
        registry_credentials = local.registry_credentials
      }
    }
  }
}
