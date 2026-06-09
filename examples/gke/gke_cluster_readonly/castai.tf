# 3. Connect GKE cluster to CAST AI in read-only mode.

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

# Configure GKE cluster connection to CAST AI in read-only mode.
resource "castai_gke_cluster" "this" {
  project_id = var.project_id
  location   = module.gke.location
  name       = var.cluster_name
}

resource "helm_release" "castai_agent" {
  name             = "castai-agent"
  repository       = "https://castai.github.io/helm-charts"
  chart            = "castai-agent"
  namespace        = "castai-agent"
  create_namespace = true
  cleanup_on_fail  = true

  set = concat(
    [
      {
        name  = "provider"
        value = "gke"
      },
      {
        # Required until https://github.com/castai/helm-charts/issues/135 is fixed.
        name  = "createNamespace"
        value = "false"
      },
    ],
    var.castai_api_url != "" ? [{
      name  = "apiURL"
      value = var.castai_api_url
    }] : [],
  )

  set_sensitive = [
    {
      name  = "apiKey"
      value = castai_gke_cluster.this.cluster_token
    },
  ]
}
