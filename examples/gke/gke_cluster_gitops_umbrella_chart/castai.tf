resource "castai_gke_cluster" "this" {
  project_id                 = var.project_id
  location                   = var.cluster_region
  name                       = var.cluster_name
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  credentials_json = module.castai-gke-iam.private_key
}

module "castai-gke-iam" {
  source  = "castai/gke-iam/castai"
  version = "~> 0.5"

  project_id       = var.project_id
  gke_cluster_name = var.cluster_name
}
