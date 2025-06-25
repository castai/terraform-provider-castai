# Configure Data sources and providers required for CAST AI connection.
provider "castai" {
  api_token = var.castai_api_token
  api_url   = var.castai_api_url
}

provider "helm" {
  kubernetes = {
    host                   = "https://${module.gke.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  }
}
