provider "castai" {
  api_token = var.castai_api_token
}

provider "azurerm" {
  features {}
}

data "azurerm_subscription" "current" {}

provider "azuread" {
  tenant_id = data.azurerm_subscription.current.tenant_id
}

module "castai-aks-iam" {
  source = "../../../terraform-castai-aks-iam"

  aks_cluster_name        = azurerm_kubernetes_cluster.example.name
  aks_node_resource_group = azurerm_kubernetes_cluster.example.node_resource_group
  aks_resource_group      = azurerm_kubernetes_cluster.example.resource_group_name
}

module "castai-aks-cluster" {
  source = "../../../terraform-castai-aks-cluster"

  aks_cluster_name    = var.aks_cluster_name
  aks_cluster_region  = var.aks_cluster_region
  node_resource_group = azurerm_kubernetes_cluster.example.node_resource_group

  delete_nodes_on_disconnect = true

  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id
  client_id       = module.castai-aks-iam.aks_client_id
  client_secret   = module.castai-aks-iam.aks_client_secret
}
