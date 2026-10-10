locals {
  secret_env = {
    JWT_SECRET      = var.jwt_secret
    ENCRYPTION_KEY  = var.encryption_key
    LICENSE_KEY     = var.license_key
    BOOTSTRAP_TOKEN = var.bootstrap_token

    SES_ACCESS_KEY_ID     = var.ses_access_key_id
    SES_SECRET_ACCESS_KEY = var.ses_secret_access_key
  }

  # Connection budget. A 1 GB Postgres plan allows 22 backend connections, and every process opens
  # two pools: database/sql (DB_MAX_OPEN_CONNS: ent, the event bus and the job workers' own
  # queries) and river's pgx pool (PGX_MAX_CONNS: river's fetch, completion, LISTEN and leader
  # election queries). One process peaks at db_max_open_conns + pgx_max_conns = 5 + 5 = 10.
  #
  # river runs 20 workers (default 5, broadcasts 10, webhooks 5; fixed in code, not configurable),
  # but a worker does its database work through the database/sql pool and holds no pgx connection
  # (no river transactions), so the pgx pool does NOT need one connection per worker. The price of
  # 5: job completions queue briefly under load, and 20 workers share the 5 sql connections.
  #
  #   steady state    1 service                              10
  #   deploy          old service + migrate job              10 + ~4 (goose: sql.Open, uncapped but
  #                   (the job runs before the new instance;  one statement at a time = 1-2; river's
  #                    it overlaps only the OLD one)          migrator pool, same: 1-2)
  #   then            old + new service                      10 + 10 = 20 of 22 (2 for admin)
  #
  # The new instance starts only after the PRE_DEPLOY job exits, so the job never overlaps both.
  # A third process (3 x 10 = 30) would NOT fit: a second service instance needs smaller pools.
  plain_env = {
    APP_URL           = "https://${local.app_host}"
    PORT              = tostring(var.app_port)
    OTEL_SERVICE_NAME = var.otel_service_name
    DB_MAX_OPEN_CONNS = tostring(var.db_max_open_conns)
    PGX_MAX_CONNS     = tostring(var.pgx_max_conns)

    SYSTEM_EMAIL_PROVIDER = "ses"
    SYSTEM_EMAIL_FROM     = local.system_email_from
    SES_REGION            = var.ses_region
  }

  registry_credentials = var.registry_credentials != "" ? var.registry_credentials : null

  system_email_from = coalesce(var.system_email_from, "noreply@${var.domain}")

  image = {
    registry_type        = "GHCR"
    registry             = var.image_registry
    repository           = var.image_repository
    tag                  = var.image_tag
    registry_credentials = local.registry_credentials
  }

  app_host     = "${var.app_host_label}.${var.domain}"
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
    # dns.tf, and it issues a TLS certificate for each name. The app host is the primary domain. The
    # apex is reserved for the marketing site hosted elsewhere: the app creates no apex record.
    domain {
      name = local.app_host
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

      # The app host serves the SPA and /site/* (no rewrite). The apex has no rule on purpose.
      rule {
        match {
          authority {
            exact = local.app_host
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

    # Neither component sets an entrypoint: the image has ENTRYPOINT ["sphericon"] (binary at
    # /usr/local/bin/sphericon, on PATH) and no CMD, so the service runs `sphericon` = the server.
    # One service: HTTP, river and watermill all run in this process, so there is no worker.
    service {
      name               = "web"
      instance_size_slug = "apps-s-1vcpu-1gb"
      instance_count     = 1
      http_port          = var.app_port

      dynamic "image" {
        for_each = [local.image]
        content {
          registry_type        = image.value.registry_type
          registry             = image.value.registry
          repository           = image.value.repository
          tag                  = image.value.tag
          registry_credentials = image.value.registry_credentials
        }
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
    # run_command replaces the image ENTRYPOINT (DigitalOcean: "For Dockerfile-based builds,
    # entering a run command overrides the Dockerfile's entrypoint",
    # https://docs.digitalocean.com/products/app-platform/how-to/deploy-from-container-images/),
    # so the full command is `sphericon migrate`. The docs do not say what happens to CMD (there is
    # none here). Confirm in the first live deploy (README troubleshooting).
    job {
      name               = "migrate"
      kind               = "PRE_DEPLOY"
      instance_size_slug = "apps-s-1vcpu-0.5gb"
      instance_count     = 1
      run_command        = "sphericon migrate"

      dynamic "image" {
        for_each = [local.image]
        content {
          registry_type        = image.value.registry_type
          registry             = image.value.registry
          repository           = image.value.repository
          tag                  = image.value.tag
          registry_credentials = image.value.registry_credentials
        }
      }
    }
  }
}
