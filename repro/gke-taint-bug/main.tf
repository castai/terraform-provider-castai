# Minimal repro for the castai_gke_cluster taint->reapply bug.
#
# The castai_gke_cluster resource connects an EXISTING GKE cluster to CAST AI via
# ExternalClusterAPIRegisterCluster (Create) and ExternalClusterAPIDisconnectCluster /
# ExternalClusterAPIDeleteCluster (Delete). The bug under investigation: after
# `terraform taint castai_gke_cluster.this` + `terraform apply`, terraform reports
# apply complete but the CAST AI cluster ends up disconnected/deleted/archived.
#
# This config targets exactly one real GKE cluster:
#   name        = furkhatk-2007
#   project_id  = engineering-test-353509
#   location    = europe-central2-a   (zonal)
#
# CAST AI credentials are sourced from the repo .env via environment variables.
# run-repro.sh sources ~/castai/terraform-provider-castai/.env (which exports
# CASTAI_API_URL and CASTAI_API_TOKEN) and also exports the TF_VAR_* mirrors so the
# variables below are populated. The provider's own EnvDefaultFunc would also pick
# CASTAI_API_URL / CASTAI_API_TOKEN up directly if the variables were unset, but we
# pass them explicitly so the wiring is visible in the config.

variable "castai_api_url" {
  type        = string
  description = "CAST AI API URL. Populated by run-repro.sh from CASTAI_API_URL in ~/castai/terraform-provider-castai/.env."
}

variable "castai_api_token" {
  type        = string
  sensitive   = true
  description = "CAST AI API token. Populated by run-repro.sh from CASTAI_API_TOKEN in ~/castai/terraform-provider-castai/.env."
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

# The single resource under test. credentials_json is intentionally omitted: it is
# optional (only used by updateGKEClusterSettings) and the Create/Register path does
# not require it. Omitting it keeps the repro minimal and avoids embedding GCP
# service-account secrets in the repro directory.
resource "castai_gke_cluster" "this" {
  name       = "furkhatk-2007"
  project_id = "engineering-test-353509"
  location   = "europe-central2-a"
}

output "cluster_id" {
  value       = castai_gke_cluster.this.id
  description = "CAST AI cluster id assigned by ExternalClusterAPIRegisterCluster. Used by check-status.sh and the cluster-timeline poller."
}
