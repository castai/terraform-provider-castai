module "castai-gke-iam" {
  source  = "castai/gke-iam/castai"
  version = "~> 0.6"

  project_id                  = var.project_id
  gke_cluster_name            = var.cluster_name
  authentication_method       = var.authentication_method
  service_accounts_unique_ids = length(var.service_accounts_unique_ids) == 0 ? [] : var.service_accounts_unique_ids
}
