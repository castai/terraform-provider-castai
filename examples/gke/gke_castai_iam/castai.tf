module "castai-gke-iam" {
  source = "castai/gke-iam/castai"

  project_id                  = var.project_id
  gke_cluster_name            = var.cluster_name
  service_accounts_unique_ids = length(var.service_accounts_unique_ids) == 0 ? [] : var.service_accounts_unique_ids
}