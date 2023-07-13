resource "castai_aks_cluster" "this" {
  name = var.cluster_name

  region          = var.cluster_region
  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id
  client_id       = azuread_application.castai.application_id
  client_secret   = azuread_application_password.castai.value


  node_resource_group        = data.azurerm_kubernetes_cluster.example.node_resource_group
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
}

resource "castai_node_configuration" "default" {
  cluster_id     = castai_aks_cluster.this.id
  name           = "default"
  disk_cpu_ratio = 0
  min_disk_size  = 100
  subnets        = var.subnets

  aks {
    max_pods_per_node = 40
  }
}

resource "castai_node_configuration_default" "this" {
  cluster_id       = castai_aks_cluster.this.id
  configuration_id = castai_node_configuration.default.id
}
