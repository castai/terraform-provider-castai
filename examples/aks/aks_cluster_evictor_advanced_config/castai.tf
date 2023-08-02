# 3. Connect AKS cluster to CAST AI in READ-ONLY mode.

# Configure Data sources and providers required for CAST AI connection.
data "azurerm_subscription" "current" {}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes {
    host                   = azurerm_kubernetes_cluster.this.kube_config.0.host
    client_certificate      = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.client_certificate)
    client_key             = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.client_key)
    cluster_ca_certificate  = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.cluster_ca_certificate)
  }
}

resource "kubernetes_config_map" "castai_evictor_config" {
  metadata {
    name   = "castai-evictor-config"
    labels = {
      "app.kubernetes.io/name" = "castai-evictor-config"
    }
  }
  data = var.evictor_advanced_config
}
