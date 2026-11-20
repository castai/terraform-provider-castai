# Connects the cluster to CAST AI.
# Notice that URLs are the same public CAST AI names as usual.
# See privatelink.tf where these names are redirected to the Private link service.
# From there on, communication works as-if the workloads were reaching CAST AI normally.
module "castai-aks-cluster" {
  source  = "castai/aks/castai"

  api_url          = var.castai_api_url
  grpc_url         = var.castai_grpc_url
  kvisor_grpc_addr = var.castai_kvisor_grpc_addr

  castai_api_token       = var.castai_api_token
  wait_for_cluster_ready = true

  authentication_method = "workload_identity"

  workload_autoscaler_keep_crds = true
  umbrella_enabled              = true

  aks_cluster_name    = azurerm_kubernetes_cluster.this.name
  aks_cluster_region  = var.cluster_region
  node_resource_group = azurerm_kubernetes_cluster.this.node_resource_group
  resource_group      = azurerm_kubernetes_cluster.this.resource_group_name

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  subscription_id = data.azurerm_subscription.current.subscription_id
  tenant_id       = data.azurerm_subscription.current.tenant_id

  default_node_configuration = module.castai-aks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      disk_cpu_ratio = 25
      subnets        = [azurerm_subnet.aks_nodes.id]
      tags           = local.tags
    }
  }

  # Start the agents only once the private endpoint and its DNS records exist, so
  # the first API call already takes the private path instead of the public one
  # (both work in this example, only one is private).
  depends_on = [
    azurerm_private_endpoint.castai,
    azurerm_private_dns_zone_virtual_network_link.castai,
    azurerm_private_dns_a_record.castai,
  ]
}
