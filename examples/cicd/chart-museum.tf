resource "cloudflare_record" "charts" {
  zone_id = lookup(data.cloudflare_zones.cicd.zones[0], "id")
  name    = var.cloudflare_charts_subdomain
  value   = data.kubernetes_service.ingress.status.0.load_balancer.0.ingress.0.hostname
  type    = "CNAME"
  ttl     = 300
  proxied = false
}

resource "kubernetes_namespace" "chart_museum" {
  metadata {
    name = "chart-museum"
  }
}

resource "kubernetes_secret" "chart_museum_basic_auth" {
  metadata {
    name      = "chart-museum-basic-auth"
    namespace = kubernetes_namespace.chart_museum.metadata[0].name
  }
  data = {
    user = var.charts_user,
    pass = var.charts_pass
  }
  depends_on = [
    kubernetes_namespace.chart_museum,
  ]
}

resource "google_storage_bucket" "chart_museum" {
  name          = "${google_project.cicd.project_id}-chart-museum"
  project       = google_project.cicd.project_id
  location      = var.gcp_region
  storage_class = "STANDARD"
}

resource "google_service_account" "chart_museum_sa" {
  project      = google_project.cicd.project_id
  account_id   = "chart-museum"
  display_name = "chart-museum"
  description  = "SA used by Chart Museum access chart museum bucket"
}

resource "google_project_iam_member" "chart_museum_sa_iam_member" {
  provider = google-beta
  project  = google_project.cicd.project_id
  role     = "roles/storage.objectAdmin"
  member   = "serviceAccount:${google_service_account.chart_museum_sa.email}"

  condition {
    expression = <<EOE
        resource.name == "projects/_/buckets/${google_storage_bucket.chart_museum.name}" ||
        resource.name.startsWith("projects/_/buckets/${google_storage_bucket.chart_museum.name}/objects/") ||
        resource.name.startsWith("projects/_/buckets/${google_storage_bucket.chart_museum.name}/objects/_")
  EOE
    title      = "Allow access only to GCS bucket ${google_storage_bucket.chart_museum.name}"
  }
}

resource "google_service_account_key" "chart_museum_sa_key" {
  service_account_id = google_service_account.chart_museum_sa.name
}

resource "kubernetes_secret" "gcp_chart_museum_bucket_sa" {
  metadata {
    name = "gcp-chart-museum-bucket-sa"
    namespace = kubernetes_namespace.chart_museum.metadata[0].name
  }
  data = {
    "credentials.json" = base64decode(google_service_account_key.chart_museum_sa_key.private_key)
  }
}

resource "helm_release" "chart_museum" {
  name       = "chart-museum"
  namespace  = kubernetes_namespace.chart_museum.metadata[0].name
  repository = "https://chartmuseum.github.io/charts"
  chart      = "chartmuseum"
  version    = "2.16.0"
  values = [file("chart-museum-values.yaml")]
  depends_on = [
    google_storage_bucket.chart_museum,
    kubernetes_namespace.chart_museum,
    kubernetes_secret.chart_museum_basic_auth,
    cloudflare_record.charts
  ]

  set {
    name  = "env.open.STORAGE"
    value = "google"
  }
  set {
    name  = "env.open.STORAGE_GOOGLE_BUCKET"
    value = google_storage_bucket.chart_museum.name
  }
  set {
    name  = "gcp.secret.name"
    value = kubernetes_secret.gcp_chart_museum_bucket_sa.metadata[0].name
  }
  set {
    name  = "gcp.secret.key"
    value = "credentials.json"
  }
}
