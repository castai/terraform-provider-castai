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

### Onboarding the cluster from this example

Set `onboard_cluster = true` and provide the GKE details plus credentials. Two options:

**a) Service account key** (`gke_credentials_json`):

```shell
terraform -chdir=examples/gke/node_configuration_local_ssd apply \
  -var onboard_cluster=true \
  -var gke_cluster_name=<cluster> \
  -var gke_project_id=<project> \
  -var gke_cluster_location=<zone-or-region> \
  -var gke_credentials_json="$(cat credentials.json)" \
  -var 'subnets=[<subnetwork-self-link>]'
```

**b) Service account impersonation (no exported key)** - set
`cast_service_account_email` + `target_service_account_email` instead of
`gke_credentials_json`:

```hcl
data "castai_impersonation_service_account" "gcp" {
  cloud_provider = "gcp"   # returns the per-org CAST AI jumper service account
}

# Customer-side SA (roles: see castai/gke-iam module) that CAST AI impersonates:
resource "google_service_account" "castai" {
  account_id = "castai-impersonated"
}

resource "google_service_account_iam_binding" "jumper_token_creator" {
  service_account_id = google_service_account.castai.name
  role               = "roles/iam.serviceAccountTokenCreator"
  members            = ["serviceAccount:${data.castai_impersonation_service_account.gcp.id}"]
}
```

```shell
terraform -chdir=examples/gke/node_configuration_local_ssd apply \
  -var onboard_cluster=true \
  -var gke_cluster_name=<cluster> \
  -var gke_project_id=<project> \
  -var gke_cluster_location=<zone-or-region> \
  -var cast_service_account_email=data.castai_impersonation_service_account.gcp.id \
  -var target_service_account_email=google_service_account.castai.email \
  -var 'subnets=[<subnetwork-self-link>]'
```

Notes:

- The node configuration API requires the cluster to have working CAST AI
  credentials, so the onboarding path needs either the key or the impersonation pair.
- Onboarding a cluster that is already registered with CAST AI will conflict -
  use `onboard_cluster = false` with `castai_cluster_id` in that case.
- The CAST AI agent (Helm install) is not part of this example; the
  `castai/gke-cluster/castai` registry module does the full flow (but its
  `node_configurations` map does not support `local_ssd` yet).

## Things to try

- `local_ssd_mode = "EPHEMERAL"` / `"BLOCK"` / `"MIXED"` (MIXED requires
  `ephemeral_storage_percent`; the provider rejects it for the other modes).
- Set any capacity variable (`min_local_storage_gib`, `local_storage_per_cpu_gib`,
  `min_iops`) to `0` to omit it from the request - a config without capacity fields
  is valid and means "do not care about sizing, just apply the mode".
- The API rejects `min_iops: -5` style values and out-of-range percentages with
  HTTP 400 carrying the field message.
