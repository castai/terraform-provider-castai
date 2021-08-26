// Cache bucket for gitlab runners.
resource "google_storage_bucket" "gitlab_cache" {
  name          = "${google_project.cicd.project_id}-gitlab-cache"
  project       = google_project.cicd.project_id
  location      = var.gcp_region
  storage_class = "STANDARD"
  force_destroy = true
}

resource "google_service_account" "gitlab_cache_sa" {
  project      = google_project.cicd.project_id
  account_id   = "gitlab-cache"
  display_name = "gitlab-runner-cache"
  description  = "SA used by Gitlab runner to access cache bucket"
}

resource "google_project_iam_member" "gitlab_cache_sa_iam_member" {
  provider = google-beta
  project  = google_project.cicd.project_id
  role     = "roles/storage.objectAdmin"
  member   = "serviceAccount:${google_service_account.gitlab_cache_sa.email}"

  condition {
    expression = <<EOE
        resource.name == "projects/_/buckets/${google_storage_bucket.gitlab_cache.name}" ||
        resource.name.startsWith("projects/_/buckets/${google_storage_bucket.gitlab_cache.name}/objects/") ||
        resource.name.startsWith("projects/_/buckets/${google_storage_bucket.gitlab_cache.name}/objects/_")
  EOE
    title      = "Allow access only to GCS bucket ${google_storage_bucket.gitlab_cache.name}"
  }
}

resource "google_service_account_key" "gitlab_cache_sa_key" {
  service_account_id = google_service_account.gitlab_cache_sa.name
}

resource "kubernetes_namespace" "gitlab" {
  metadata {
    name = "gitlab"
  }
}

resource "kubernetes_secret" "gcp-gitlab-cache-bucket-sa" {
  metadata {
    name      = "gcp-gitlab-cache-bucket-sa"
    namespace = kubernetes_namespace.gitlab.metadata[0].name
  }
  data = {
    "gcs-applicaton-credentials-file" = base64decode(google_service_account_key.gitlab_cache_sa_key.private_key)
  }
}

// Helm chart.
resource "helm_release" "gitlab-runner" {
  name       = "gitlab-runner"
  repository = "https://charts.gitlab.io"
  namespace  = kubernetes_namespace.gitlab.metadata[0].name
  chart      = "gitlab-runner"
  version    = "0.27.0"
  values     = [file("gitlab-runners-values.yaml")]
  depends_on = [
    google_storage_bucket.gitlab_cache,
    kubernetes_secret.gcp-gitlab-cache-bucket-sa
  ]

  set {
    name  = "runnerRegistrationToken"
    value = var.gitlab_runner_registration_token
  }

  set {
    name  = "runners.cache.gcsBucketName"
    value = google_storage_bucket.gitlab_cache.name
  }

  set {
    name  = "runners.cache.secretName"
    value = kubernetes_secret.gcp-gitlab-cache-bucket-sa.metadata[0].name
  }
}
