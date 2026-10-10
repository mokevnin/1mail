#!/usr/bin/env bash
# Shared by the infra:* tasks (sourced). doctl is the single credential source: its token drives
# the DigitalOcean provider, and a DigitalOcean Secrets Manager container holds every other
# secret. Nothing is written to disk and no secret is passed in argv; values travel only through
# pipes and the environment. No function here ever prints a value.
set -euo pipefail
set +x

INFRA_SECRETS_NAME="${INFRA_SECRETS_NAME:-sphericon-infra}"
INFRA_REGION="${INFRA_REGION:-fra1}"
INFRA_STATE_BUCKET="${INFRA_STATE_BUCKET:-sphericon-tfstate}"
INFRA_STATE_KEY="${INFRA_STATE_KEY:-production/terraform.tfstate}"

die() {
  echo "error: $*" >&2
  exit 1
}

require_doctl() {
  command -v doctl >/dev/null || die "doctl is not installed (mise install)"
  doctl auth token >/dev/null 2>&1 || die "doctl is not authenticated: run 'doctl auth init'"
}

# secret_exists: true when the container is listed (list never returns values).
secret_exists() {
  doctl secrets list --no-header --format Name | grep -Fxq "$INFRA_SECRETS_NAME"
}

# secret_put <create|set>: reads KEY=VALUE lines from stdin into the container. The env-file path
# is /dev/stdin, so neither the values nor a temp file appear anywhere.
secret_put() {
  doctl secrets "$1" "$INFRA_SECRETS_NAME" --region "$INFRA_REGION" --from-env-file /dev/stdin >/dev/null
}

# secret_export <KEY>...: exports each named container key as an env var of the same name; a
# missing key is left unset. Values are split on the first "=" and never evaluated.
secret_export() {
  local kvs line key want
  kvs="$(doctl secrets get "$INFRA_SECRETS_NAME" --region "$INFRA_REGION" --kvs)" ||
    die "cannot read secret '$INFRA_SECRETS_NAME' (run: mise run infra:bootstrap)"
  while IFS= read -r line; do
    key="${line%%=*}"
    for want in "$@"; do
      if [[ "$key" == "$want" ]]; then
        export "$key=${line#*=}"
      fi
    done
  done <<<"$kvs"
}
