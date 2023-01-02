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

module "castai-aks-cluster" {
  source = "castai/aks/castai"

  aks_cluster_name    = var.aks_cluster_name
  aks_cluster_region  = var.aks_cluster_region
  node_resource_group = azurerm_kubernetes_cluster.this.node_resource_group
  resource_group      = azurerm_kubernetes_cluster.this.resource_group_name

  delete_nodes_on_disconnect = true

  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id

  default_node_configuration = module.castai-aks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = [azurerm_subnet.internal.id]
      tags           = {
        "env" : "prod"
      }
    }
  }
}
