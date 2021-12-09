## IAM user required for CAST.AI

terraform {
  required_providers {
    castai = {
      source  = "castai/castai"
      version = "0.0.9-local"
    }
  }
  required_version = ">= 0.13"
}

provider "castai" {
  api_token = var.castai_api_token
  api_url = var.castai_url
}

provider "google" {
  project     = var.project_id
  region      = "eu-central1"
}

locals {
  service_account_email = "${local.service_account_id}@${var.project_id}.iam.gserviceaccount.com"
  service_account_id    =  "castai-gke-${substr(sha1(var.cluster_name),1,8)}"
}

resource "google_service_account" "castai" {
  account_id   = local.service_account_id
  display_name = "Service account to manage ${var.cluster_name} cluster via CAST"
  project      = var.project_id
}

data "castai_gcp_user_policies" "gke" {}

resource "google_project_iam_custom_role" "castai_role" {
  role_id     = var.custom_role_id
  title       = "Role to manage GKE cluster via CAST AI"
  description = "Role to manage GKE cluster via CAST AI"
  permissions = toset(data.castai_gcp_user_policies.gke.policy)
  project     = var.project_id
  stage       = "ALPHA"
}

resource "google_project_iam_binding" "project" {
  for_each = toset([
    "roles/container.developer",
    "roles/iam.serviceAccountUser",
    "projects/${var.project_id}/roles/${var.custom_role_id}"
  ])

  project = var.project_id
  role    = each.key
  members = [local.service_account_email]
}

resource "google_service_account_key" "castai_key" {
  service_account_id = local.service_account_id
  public_key_type    = "TYPE_X509_PEM_FILE"
}

