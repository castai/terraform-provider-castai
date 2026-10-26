data "castai_impersonation_service_account" "this" {
  provider_type = "gke"
}

resource "castai_gke_cluster" "this" {
  project_id                 = var.project_id
  location                   = module.gke.location
  name                       = var.gke_cluster_name
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  credentials {
    target_service_account_email = google_service_account.target.email
    castai_service_account_email = data.castai_impersonation_service_account.this.email
  }
}
