# Following providers required by AKS and Vnet resources.
provider "azurerm" {
  features {}
  subscription_id = var.subscription_id # From Azure version 4.0, Specifying Subscription ID is Mandatory
}

provider "azuread" {
  tenant_id = data.azurerm_subscription.current.tenant_id
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_key
}

provider "helm" {
  kubernetes {
    host                   = data.azurerm_kubernetes_cluster.example.kube_config.0.host
    client_certificate     = base64decode(data.azurerm_kubernetes_cluster.example.kube_config.0.client_certificate)
    client_key             = base64decode(data.azurerm_kubernetes_cluster.example.kube_config.0.client_key)
    cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.example.kube_config.0.cluster_ca_certificate)
  }
}
