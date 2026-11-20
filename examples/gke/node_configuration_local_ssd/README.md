# Verifying the `gke.local_ssd` block in practice

This example creates a `castai_node_configuration` with the new GKE `local_ssd`
storage recipe on an already onboarded GKE cluster. Applying it only registers the
configuration - nodes with local SSDs are provisioned only if the autoscaler picks
this configuration.

## 1. Build this branch of the provider

```shell
make build   # produces ./terraform-provider-castai in the repository root
```

## 2. Use the local build via dev overrides

```shell
sed "s|REPO_ROOT|$(git rev-parse --show-toplevel)|" dev.tfrc.example > /tmp/dev.tfrc
export TF_CLI_CONFIG_FILE=/tmp/dev.tfrc
```

With `dev_overrides` Terraform skips provider installation and uses the freshly
built binary (no `terraform init` needed).

## 3. Point the provider at your API

```shell
# Local Tilt stack (needs the stack running):
export CASTAI_API_URL=https://api--<your-machine>.local.castai.co
export CASTAI_API_TOKEN=<token for that environment>
# Against production, unset CASTAI_API_URL (defaults to https://api.cast.ai).
```

## 4. Apply

```shell
terraform -chdir=examples/gke/node_configuration_local_ssd init -backend=false
terraform -chdir=examples/gke/node_configuration_local_ssd apply \
  -var castai_cluster_id=<uuid> \
  -var 'subnets=[<subnetwork-self-link>]'
```

## Things to try

- `local_ssd_mode = "EPHEMERAL"` / `"BLOCK"` / `"MIXED"` (MIXED requires
  `ephemeral_storage_percent`; the provider rejects it for the other modes).
- Set any capacity variable (`min_local_storage_gib`, `local_storage_per_cpu_gib`,
  `min_iops`) to `0` to omit it from the request - a config without capacity fields
  is valid and means "do not care about sizing, just apply the mode".
- The API rejects `min_iops: -5` style values and out-of-range percentages with
  HTTP 400 carrying the field message.
