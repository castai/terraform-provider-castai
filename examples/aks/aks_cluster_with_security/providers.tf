# Following providers required by AKS and Vnet resources.
provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
}

provider "castai" {
  api_token = var.castai_api_token
  api_url   = var.castai_api_url
}

provider "azuread" {
  tenant_id = data.azurerm_subscription.current.tenant_id
}

provider "helm" {
  kubernetes {
    host                   = azurerm_kubernetes_cluster.this.kube_config.0.host
    client_certificate     = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.client_certificate)
    client_key             = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.client_key)
    cluster_ca_certificate = base64decode(azurerm_kubernetes_cluster.this.kube_config.0.cluster_ca_certificate)
  }
}
