locals {
  role_name = "CastAKSRole-${var.cluster_name}-tf"
  app_name  = substr("CAST AI ${var.cluster_name}-${var.resource_group}", 0, 64)
}

data "azuread_client_config" "current" {}

data "azurerm_subscription" "current" {}

data "azurerm_kubernetes_cluster" "this" {
  name                = var.cluster_name
  resource_group_name = var.resource_group
}

