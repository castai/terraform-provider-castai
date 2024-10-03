locals {
  service_account_id = "castai-gke-tf-${substr(sha1(var.cluster_name), 0, 8)}"
}

data "castai_gke_user_policies" "gke" {}

data "google_project" "project" {
  project_id = var.project_id
}

resource "google_service_account" "client_service_account" {
  account_id   = local.service_account_id
  display_name = "Service account to manage ${var.cluster_name} cluster via CAST"
  project      = var.project_id
}

resource "google_project_iam_custom_role" "castai_role" {
  role_id     = "castai.gkeAccess.${substr(sha1(var.cluster_name), 0, 8)}.tf"
  title       = "Role to manage GKE cluster via CAST AI"
  description = "Role to manage GKE cluster via CAST AI"
  permissions = toset(data.castai_gke_user_policies.gke.policy)
  project     = var.project_id
  stage       = "GA"
}

resource "google_project_iam_binding" "compute_manager_binding" {
  project = var.project_id
  role    = "projects/${var.project_id}/roles/castai.gkeAccess.${substr(sha1(var.cluster_name), 0, 8)}.tf"
  members = ["serviceAccount:${google_service_account.client_service_account.email}"]
}

# Configure GKE cluster and obtain the castai service account.
resource "castai_gke_cluster_id" "cluster_id" {
  name                   = var.cluster_name
  location               = var.cluster_region
  project_id             = var.project_id
  client_service_account = google_service_account.client_service_account.email
  # DO NOT UNCOMMENT: cast service account will be computed and filled after apply.
  # cast_service_account   = "to-be-computed"
}

# Grant the roles/iam.serviceAccountTokenCreator role to the CASTAI_SERVICE_ACCOUNT
resource "google_service_account_iam_member" "token_creator_binding" {
  service_account_id = google_service_account.client_service_account.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${castai_gke_cluster_id.cluster_id.cast_service_account}"

  condition {
    title       = "AlwaysTrueCondition"
    description = "This condition is always true"
    expression  = "true"
  }

  depends_on = [castai_gke_cluster_id.cluster_id]
}

# Grant the roles/iam.serviceAccountUser role to the CASTAI_SERVICE_ACCOUNT with a specific condition
resource "google_service_account_iam_member" "impersonation_user_binding" {
  service_account_id = google_service_account.client_service_account.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${castai_gke_cluster_id.cluster_id.cast_service_account}"

  condition {
    title       = "SpecificServiceAccountCondition"
    description = "Allow impersonation only for CASTAI_SERVICE_ACCOUNT"
    expression  = "request.auth.claims.email == \"${castai_gke_cluster_id.cluster_id.cast_service_account}\""
  }

  depends_on = [castai_gke_cluster_id.cluster_id]
}

# Grant the roles/iam.serviceAccountUser role to the CLIENT_SERVICE_ACCOUNT without.
resource "google_project_iam_member" "service_account_user" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.client_service_account.email}"
}

// service_account
resource "time_sleep" "wait_3_minutes" {
  depends_on = [
    google_service_account.client_service_account,
    google_service_account_iam_member.token_creator_binding,
    google_service_account_iam_member.impersonation_user_binding,
    google_project_iam_member.service_account_user,
  ]

  create_duration = "180s"
}