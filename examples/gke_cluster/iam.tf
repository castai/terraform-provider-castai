## IAM user required for CAST.AI

provider "castai" {
  api_token = var.castai_api_token
}

provider "google" {
  project     = var.project_id
  region      = var.cluster_region
}

locals {
  service_account_id    = "castai-gke-tf-${substr(sha1(var.cluster_name),0,8)}"
  service_account_email = "${local.service_account_id}@${var.project_id}.iam.gserviceaccount.com"
  custom_role_id        = "castai.gkeAccess.tf"
}

resource "google_service_account" "castai_service_account" {
  account_id   = local.service_account_id
  display_name = "Service account to manage ${var.cluster_name} cluster via CAST"
  project      = var.project_id
}

data "castai_gke_user_policies" "gke" {}

resource "google_project_iam_custom_role" "castai_role" {
  role_id     = local.custom_role_id
  title       = "Role to manage GKE cluster via CAST AI"
  description = "Role to manage GKE cluster via CAST AI"
  permissions = toset(data.castai_gke_user_policies.gke.policy)
  project     = var.project_id
  stage       = "ALPHA"
}

resource "google_project_iam_binding" "project" {
  for_each = toset([
    "roles/container.developer",
    "roles/iam.serviceAccountUser",
    "projects/${var.project_id}/roles/${local.custom_role_id}"
  ])

  project = var.project_id
  role    = each.key
  members = ["serviceAccount:${local.service_account_email}"]
}

resource "google_service_account_key" "castai_key" {
  service_account_id = google_service_account.castai_service_account.account_id
  public_key_type    = "TYPE_X509_PEM_FILE"
}

output "private_key" {
  value = base64decode(google_service_account_key.castai_key.private_key)
  sensitive = true
}
