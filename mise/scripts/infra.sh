#!/usr/bin/env bash
# Run Terraform in infra/ with every credential taken from doctl and its Secrets Manager
# container (mise run infra -- plan|apply|destroy|init|...). Nothing is read from files and
# nothing is printed. State access: the Spaces key is exported as AWS_ACCESS_KEY_ID, which the s3
# backend reads; the aws provider (SES) gets its own key through variables instead
# (TF_VAR_aws_access_key_id), see infra/providers.tf.
set -euo pipefail
set +x
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=infra-common.sh
source "$here/infra-common.sh"

infra_dir="$here/../../infra"

# Filled by secret_export from the container (declared so the reads below are visible to lint).
SES_ACCESS_KEY_ID="" SES_SECRET_ACCESS_KEY=""

require_doctl
secret_export SPACES_ACCESS_KEY_ID SPACES_SECRET_ACCESS_KEY ENCRYPTION_KEY JWT_SECRET \
  BOOTSTRAP_TOKEN LICENSE_KEY REGISTRY_CREDENTIALS SES_ACCESS_KEY_ID SES_SECRET_ACCESS_KEY \
  TF_AWS_ACCESS_KEY_ID TF_AWS_SECRET_ACCESS_KEY

required=(SPACES_ACCESS_KEY_ID SPACES_SECRET_ACCESS_KEY ENCRYPTION_KEY JWT_SECRET BOOTSTRAP_TOKEN
  REGISTRY_CREDENTIALS SES_ACCESS_KEY_ID SES_SECRET_ACCESS_KEY TF_AWS_ACCESS_KEY_ID
  TF_AWS_SECRET_ACCESS_KEY)
missing=()
for name in "${required[@]}"; do
  [[ -n "${!name:-}" ]] || missing+=("$name")
done
((${#missing[@]} == 0)) || die "missing in '$INFRA_SECRETS_NAME': ${missing[*]} (mise run infra:secret NAME)"

DIGITALOCEAN_TOKEN="$(doctl auth token)"
export DIGITALOCEAN_TOKEN
export AWS_ACCESS_KEY_ID="$SPACES_ACCESS_KEY_ID" AWS_SECRET_ACCESS_KEY="$SPACES_SECRET_ACCESS_KEY"
export TF_VAR_encryption_key="$ENCRYPTION_KEY" TF_VAR_jwt_secret="$JWT_SECRET"
export TF_VAR_bootstrap_token="$BOOTSTRAP_TOKEN" TF_VAR_license_key="${LICENSE_KEY:-}"
export TF_VAR_registry_credentials="$REGISTRY_CREDENTIALS"
export TF_VAR_ses_access_key_id="$SES_ACCESS_KEY_ID"
export TF_VAR_ses_secret_access_key="$SES_SECRET_ACCESS_KEY"
export TF_VAR_aws_access_key_id="$TF_AWS_ACCESS_KEY_ID"
export TF_VAR_aws_secret_access_key="$TF_AWS_SECRET_ACCESS_KEY"
export TF_INPUT=0

tf=(terraform -chdir="$infra_dir")
backend=(-backend-config="bucket=$INFRA_STATE_BUCKET" -backend-config="key=$INFRA_STATE_KEY")

cmd="${1:-}"
[[ -n "$cmd" ]] || die "usage: mise run infra -- init|plan|apply|destroy|output ..."
shift

if [[ "$cmd" == init ]]; then
  exec "${tf[@]}" init "${backend[@]}" "$@"
fi
# Initialise by itself when the s3 backend is not configured yet.
if ! grep -qs '"backend"' "$infra_dir/.terraform/terraform.tfstate"; then
  "${tf[@]}" init "${backend[@]}" >&2
fi

# Non-secret values come from infra/production.tfvars (gitignored). `apply <planfile>` takes no
# -var-file: a first argument that is not a flag is a saved plan.
vars=()
case "$cmd" in
  plan | destroy | import | refresh) vars=(-var-file=production.tfvars) ;;
  apply) [[ "${1:-}" == -* || $# -eq 0 ]] && vars=(-var-file=production.tfvars) ;;
  *) ;;
esac
exec "${tf[@]}" "$cmd" ${vars[@]+"${vars[@]}"} "$@"
