resource "castai_gke_cluster" "this" {
  project_id                 = var.project_id
  location                   = module.gke.location
  name                       = var.gke_cluster_name
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
  credentials_json           = var.gke_credentials
}