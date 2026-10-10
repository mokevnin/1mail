# Sphericon infrastructure (Terraform)

The hosted Sphericon deployment is described only here (ADR 0028). Nothing is created by hand in
the DigitalOcean console or through an MCP server, except the one-time state bucket below.

## Checks

`mise run check:infra` runs `terraform fmt -check`, `init -backend=false` and `validate`. It needs no
credentials and no network state; CI and the git hook run the same task.

## One-time bootstrap: the state bucket

State lives in a private DigitalOcean Spaces bucket, which Terraform cannot create for itself.
`doctl` has no bucket-create command, so use the S3 API (or the console).

1. Create a full-access Spaces key (the secret is shown once):

   ```sh
   doctl spaces keys create sphericon-bootstrap --grants 'bucket=;permission=fullaccess'
   ```

2. Map it to the AWS variable names the S3 tooling and Terraform backend read:

   ```sh
   export AWS_ACCESS_KEY_ID=<SPACES access key>
   export AWS_SECRET_ACCESS_KEY=<SPACES secret key>
   ```

3. Create the bucket, private (bucket names are global across Spaces, so pick a unique one):

   ```sh
   aws s3api create-bucket --bucket sphericon-tfstate --acl private \
     --endpoint-url https://fra1.digitaloceanspaces.com
   ```

4. Verify nothing is public: the ACL must have no `AllUsers` grant.

   ```sh
   aws s3api get-bucket-acl --bucket sphericon-tfstate \
     --endpoint-url https://fra1.digitaloceanspaces.com
   ```

5. Narrow the access: create a key scoped to that bucket only and delete the bootstrap key.

   ```sh
   doctl spaces keys create sphericon-tfstate --grants 'bucket=sphericon-tfstate;permission=readwrite'
   doctl spaces keys delete sphericon-bootstrap
   ```

   Export the new key as `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`.

## Working with the module

The provider token comes from the environment, never from a file:

```sh
export DIGITALOCEAN_TOKEN=<API token>
terraform -chdir=infra init \
  -backend-config="bucket=sphericon-tfstate" \
  -backend-config="key=production/terraform.tfstate"
```

Or copy `backend.hcl.example` to `backend.hcl` (gitignored) and pass `-backend-config=backend.hcl`.
State holds sensitive variables, so the bucket stays private; `*.tfstate*`, `*.tfvars` and
`backend.hcl` are gitignored.
