module "castai-gke-iam" {
  source  = "castai/gke-iam/castai"
  version = "~> 0.5"

  project_id                  = var.project_id
  gke_cluster_name            = var.cluster_name
  service_accounts_unique_ids = length(var.service_accounts_unique_ids) == 0 ? [] : var.service_accounts_unique_ids
}