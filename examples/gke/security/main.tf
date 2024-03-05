provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "google" {
  credentials = base64decode(var.gcp_credentials)
  region      = var.cluster_region
}

data "google_client_config" "config" {}

data "google_service_account" "account" {
  account_id = var.service_account_id
}

data "google_container_cluster" "cluster" {
  name     = var.cluster_name
  location = var.cluster_region
  project  = var.project_id
}

provider "helm" {
  kubernetes {
    host                   = "https://${data.google_container_cluster.cluster.endpoint}"
    token                  = data.google_client_config.config.access_token
    cluster_ca_certificate = base64decode(data.google_container_cluster.cluster.master_auth.0.cluster_ca_certificate)
  }
}

resource "google_service_account_key" "key" {
  service_account_id = data.google_service_account.account.name
}

resource "castai_gke_cluster" "cluster" {
  location         = data.google_container_cluster.cluster.location
  name             = data.google_container_cluster.cluster.name
  project_id       = data.google_container_cluster.cluster.project
  credentials_json = google_service_account_key.key.private_key
}

resource "helm_release" "castai_agent" {
  chart            = "castai-agent"
  name             = "castai-agent"
  repository       = "https://castai.github.io/helm-charts"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true

  set {
    name  = "provider"
    value = "gke"
  }

  set_sensitive {
    name  = "apiKey"
    value = castai_gke_cluster.cluster.cluster_token
  }

  set {
    name  = "createNamespace"
    value = "false"
  }
}

resource "helm_release" "security_agent" {
  chart            = "castai-kvisor"
  name             = "castai-kvisor"
  repository       = "https://castai.github.io/helm-charts"
  namespace        = "castai-agent"
  create_namespace = false
  cleanup_on_fail  = true
  count            = 1

  set {
    name  = "castai.apiURL"
    value = var.castai_api_url
  }

  set_sensitive {
    name  = "castai.apiKey"
    value = castai_gke_cluster.cluster.cluster_token
  }

  set {
    name  = "castai.clusterID"
    value = castai_gke_cluster.cluster.id
  }

  set {
    name  = "structuredConfig.provider"
    value = "gke"
  }

  set {
    name  = "policyEnforcement.enabled"
    value = "true"
  }
}
