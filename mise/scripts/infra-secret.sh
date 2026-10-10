#!/usr/bin/env bash
# Store one secret value in the infra container: mise run infra:secret NAME. The value is read
# from the terminal without echo (or from stdin when piped) and travels only through a pipe.
set -euo pipefail
set +x
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=infra-common.sh
source "$here/infra-common.sh"

name="${1:-}"
[[ "$name" =~ ^[A-Z][A-Z0-9_]*$ ]] || die "usage: mise run infra:secret NAME (upper case, e.g. LICENSE_KEY)"
require_doctl
secret_exists || die "secret '$INFRA_SECRETS_NAME' does not exist: run mise run infra:bootstrap"

if [[ -t 0 ]]; then
  read -r -s -p "$name: " value
  echo >&2
else
  IFS= read -r value
fi
[[ -n "$value" ]] || die "empty value"
printf '%s=%s\n' "$name" "$value" | secret_put set
echo "set $name in '$INFRA_SECRETS_NAME'"
