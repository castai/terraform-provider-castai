resource "random_string" "project_id_suffix" {
  length  = 4
  upper   = false
  special = false
}

resource "google_project" "cicd" {
  name                = var.gcp_project_name
  project_id          = "${var.gcp_project_name}-${random_string.project_id_suffix.result}"
  org_id              = var.gcp_org_id
  billing_account     = var.gcp_billing_account
  auto_create_network = false

  labels = {
    source      = "terraform"
    environment = var.gcp_project_name
  }
}

resource "google_project_service" "iam" {
  project = google_project.cicd.project_id
  service = "iam.googleapis.com"
}

resource "google_project_service" "cloudresourcemanager" {
  project = google_project.cicd.project_id
  service = "cloudresourcemanager.googleapis.com"
}

resource "google_project_service" "compute" {
  project = google_project.cicd.project_id
  service = "compute.googleapis.com"
}

resource "google_project_service" "cloudbilling" {
  project = google_project.cicd.project_id
  service = "cloudbilling.googleapis.com"
}

resource "google_service_account" "cicd" {
  project      = google_project.cicd.project_id
  account_id   = "cicd-service-account"
  display_name = "Terraform-managed service account"
}

resource "google_service_account_key" "cicd" {
  service_account_id = google_service_account.cicd.email
}

resource "google_project_iam_member" "compute" {
  project = google_project.cicd.project_id
  role    = "roles/compute.admin"
  member = "serviceAccount:${google_service_account.cicd.email}"
}

resource "google_project_iam_member" "service-account-user" {
  project = google_project.cicd.project_id
  role    = "roles/iam.serviceAccountUser"
  member = "serviceAccount:${google_service_account.cicd.email}"
}

resource "google_project_iam_member" "service-account-admin" {
  project = google_project.cicd.project_id
  role    = "roles/iam.serviceAccountAdmin"
  member = "serviceAccount:${google_service_account.cicd.email}"
}

resource "google_project_iam_member" "role-admin" {
  project = google_project.cicd.project_id
  role    = "roles/iam.roleAdmin"
  member = "serviceAccount:${google_service_account.cicd.email}"
}

resource "google_project_iam_member" "service-account-key-admin" {
  project = google_project.cicd.project_id
  role    = "roles/iam.serviceAccountKeyAdmin"
  member = "serviceAccount:${google_service_account.cicd.email}"
}

resource "google_project_iam_member" "project-iam-admin" {
  project = google_project.cicd.project_id
  role    = "roles/resourcemanager.projectIamAdmin"
  member = "serviceAccount:${google_service_account.cicd.email}"
}

// null_resource could be used if you only want to apply all gcp project related resources. Eg:
// terraform apply -target=null_resource.gcp-project
resource "null_resource" "gcp-project" {
  depends_on = [
    google_project.cicd,
    google_project_service.iam,
    google_project_service.cloudresourcemanager,
    google_project_service.compute,
    google_project_service.cloudbilling,
    google_service_account.cicd,
    google_service_account_key.cicd,
    google_project_iam_member.compute,
    google_project_iam_member.service-account-user,
    google_project_iam_member.service-account-key-admin,
    google_project_iam_member.service-account-admin,
    google_project_iam_member.project-iam-admin,
    google_project_iam_member.role-admin,
  ]
}
