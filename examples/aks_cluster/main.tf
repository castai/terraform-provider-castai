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
  source = "castai/aks-iam/castai"

  aks_cluster_name = azurerm_kubernetes_cluster.example.name
  aks_node_resource_group = azurerm_kubernetes_cluster.example.node_resource_group
  aks_resource_group = azurerm_kubernetes_cluster.example.resource_group_name
}
