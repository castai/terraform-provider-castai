data "azuread_client_config" "current" {}

data "azurerm_subscription" "current" {}

data "azurerm_kubernetes_cluster" "this" {
  name                = var.cluster_name
  resource_group_name = var.resource_group
}

