# Configure Data sources and providers required for CAST AI connection.
provider "castai" {
  api_token = var.castai_api_token
  api_url   = var.castai_api_url
}

provider "helm" {
  kubernetes {
    host                   = "https://${google_container_cluster.my-k8s-cluster.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(google_container_cluster.my-k8s-cluster.master_auth[0].cluster_ca_certificate)
  }
}