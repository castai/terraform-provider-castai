data "azurerm_subscription" "current" {}

resource "castai_aks_cluster" "this" {
  name = var.aks_cluster_name

  region          = var.aks_cluster_region
  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id
  client_id       = azuread_application.castai.application_id
  client_secret   = azuread_application_password.castai.value

  node_resource_group        = azurerm_kubernetes_cluster.this.node_resource_group
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
}