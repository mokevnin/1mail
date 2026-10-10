#!/usr/bin/env bash
# One-time, idempotent bootstrap of the hosted deployment's credentials and state bucket (ADR 0028).
# Creates, only when missing: the Secrets Manager container (Spaces key, ENCRYPTION_KEY,
# JWT_SECRET, BOOTSTRAP_TOKEN) and the private state bucket. An existing container is never
# changed, so ENCRYPTION_KEY is never rotated.
set -euo pipefail
set +x
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=infra-common.sh
source "$here/infra-common.sh"

require_doctl
for tool in jq openssl aws go; do
  command -v "$tool" >/dev/null || die "$tool is not installed (mise install)"
done

if secret_exists; then
  echo "secret '$INFRA_SECRETS_NAME' exists: keeping it"
else
  echo "creating secret '$INFRA_SECRETS_NAME' in $INFRA_REGION"
  key_json="$(doctl spaces keys create "sphericon-tfstate-$(date +%s)" \
    --grants 'bucket=;permission=fullaccess' -o json)"
  access="$(jq -r 'if type == "array" then .[0] else . end | .access_key' <<<"$key_json")"
  secret="$(jq -r 'if type == "array" then .[0] else . end | .secret_key' <<<"$key_json")"
  unset key_json
  if [[ -z "$access" || "$access" == null || -z "$secret" || "$secret" == null ]]; then
    die "doctl did not return a Spaces key"
  fi
  # The secret key is shown only once: if it cannot be stored, drop the key again.
  rollback() { doctl spaces keys delete "$access" --force >/dev/null 2>&1 || true; }
  trap rollback ERR
  encryption_key="$(cd "$here/../.." && go run ./cmd/server genkey)"
  printf '%s\n' \
    "SPACES_ACCESS_KEY_ID=$access" \
    "SPACES_SECRET_ACCESS_KEY=$secret" \
    "ENCRYPTION_KEY=$encryption_key" \
    "JWT_SECRET=$(openssl rand -hex 32)" \
    "BOOTSTRAP_TOKEN=$(openssl rand -hex 32)" | secret_put create
  trap - ERR
  unset access secret encryption_key
fi

secret_export SPACES_ACCESS_KEY_ID SPACES_SECRET_ACCESS_KEY
[[ -n "${SPACES_ACCESS_KEY_ID:-}" && -n "${SPACES_SECRET_ACCESS_KEY:-}" ]] ||
  die "secret '$INFRA_SECRETS_NAME' has no Spaces key"

# doctl cannot create buckets, so the S3 API does, with the AWS CLI. The credentials reach it
# through its environment only.
export AWS_ACCESS_KEY_ID="$SPACES_ACCESS_KEY_ID" AWS_SECRET_ACCESS_KEY="$SPACES_SECRET_ACCESS_KEY"
export AWS_DEFAULT_REGION=us-east-1 AWS_PAGER=""
s3() { aws --endpoint-url "https://${INFRA_REGION}.digitaloceanspaces.com" "$@"; }
if s3 s3api head-bucket --bucket "$INFRA_STATE_BUCKET" >/dev/null 2>&1; then
  echo "bucket '$INFRA_STATE_BUCKET' exists"
else
  echo "creating private bucket '$INFRA_STATE_BUCKET'"
  s3 s3api create-bucket --bucket "$INFRA_STATE_BUCKET" --acl private >/dev/null
fi
public="$(s3 s3api get-bucket-acl --bucket "$INFRA_STATE_BUCKET" \
  --query "Grants[?Grantee.URI=='http://acs.amazonaws.com/groups/global/AllUsers'] | length(@)" \
  --output text)"
[[ "$public" == 0 ]] || die "bucket '$INFRA_STATE_BUCKET' is publicly accessible"

cat <<MSG
Done. Next: mise run infra -- plan
Still manual (set once with: mise run infra:secret NAME): TF_AWS_ACCESS_KEY_ID,
TF_AWS_SECRET_ACCESS_KEY (Terraform's SES identity), SES_ACCESS_KEY_ID, SES_SECRET_ACCESS_KEY
(send-only IAM user), REGISTRY_CREDENTIALS (<github user>:<token>), LICENSE_KEY (optional).
MSG
