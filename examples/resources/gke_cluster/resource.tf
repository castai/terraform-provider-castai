data "castai_impersonation_service_account" "this" {
  provider_type = "gke"
}

resource "google_service_account" "target" {
  account_id   = "castai-gke"
  display_name = "CAST AI GKE cluster"
  project      = var.project_id
}

resource "google_service_account_iam_member" "token_creator" {
  service_account_id = google_service_account.target.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${data.castai_impersonation_service_account.this.email}"
}

resource "castai_gke_cluster" "this" {
  project_id                 = var.project_id
  location                   = module.gke.location
  name                       = var.gke_cluster_name
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  target_service_account_email = google_service_account.target.email
  castai_service_account_email = data.castai_impersonation_service_account.this.email
}
