resource "azurerm_user_assigned_identity" "aks" {
  name                = "${local.name}-aks-identity"
  location            = azurerm_resource_group.this.location
  resource_group_name = azurerm_resource_group.this.name
  tags                = local.tags
}

resource "azurerm_role_assignment" "aks_network_contributor" {
  scope                = azurerm_subnet.aks_nodes.id
  role_definition_name = "Network Contributor"
  principal_id         = azurerm_user_assigned_identity.aks.principal_id
  principal_type       = "ServicePrincipal"
}

resource "azurerm_kubernetes_cluster" "this" {
  name                = local.name
  location            = azurerm_resource_group.this.location
  resource_group_name = azurerm_resource_group.this.name
  dns_prefix          = local.name
  kubernetes_version  = var.aks_kubernetes_version

  sku_tier = "Free"

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.aks.id]
  }

  node_provisioning_profile {
    default_node_pools = "None"
  }

  default_node_pool {
    name            = "system"
    vm_size         = var.aks_vm_size
    node_count      = var.aks_node_count
    vnet_subnet_id  = azurerm_subnet.aks_nodes.id
    os_disk_size_gb = 64
    max_pods        = 50
  }

  network_profile {
    network_plugin      = "azure"
    network_plugin_mode = "overlay"
    pod_cidr            = local.pod_cidr
    service_cidr        = local.service_cidr
    dns_service_ip      = local.dns_service_ip
    load_balancer_sku   = "standard"

    outbound_type = "loadBalancer"
  }

  tags = local.tags

  depends_on = [azurerm_role_assignment.aks_network_contributor]
}
