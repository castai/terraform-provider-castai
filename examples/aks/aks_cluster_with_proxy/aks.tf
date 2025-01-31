
resource "azurerm_resource_group" "rg" {
  name     = var.resource_group_name
  location = var.cluster_region
}

resource "azurerm_route_table" "route_table" {
  name                = "aks-route-table"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  depends_on          = [azurerm_subnet.egress_firewall_subnet]
}

resource "azurerm_subnet_route_table_association" "route_table_association" {
  subnet_id      = azurerm_subnet.aks_subnet.id
  route_table_id = azurerm_route_table.route_table.id
}

resource "azurerm_route" "egress_route" {
  name                   = "default-egress"
  route_table_name       = azurerm_route_table.route_table.name
  resource_group_name    = azurerm_resource_group.rg.name
  address_prefix         = "0.0.0.0/0"
  next_hop_type          = "VirtualAppliance"
  next_hop_in_ip_address = azurerm_firewall.egress_firewall.ip_configuration[0].private_ip_address
}

resource "azurerm_route" "internet_route" {
  name                = "internet-egress"
  route_table_name    = azurerm_route_table.route_table.name
  resource_group_name = azurerm_resource_group.rg.name
  address_prefix      = "${azurerm_public_ip.firewall_public_ip.ip_address}/32"
  next_hop_type       = "Internet"
}

resource "azurerm_kubernetes_cluster" "aks" {
  name                = var.cluster_name
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  dns_prefix          = "aksproxy"

  default_node_pool {
    name           = "agentpool"
    vm_size        = "Standard_DS2_v2"
    node_count     = 2
    vnet_subnet_id = azurerm_subnet.aks_subnet.id

    upgrade_settings {
      drain_timeout_in_minutes      = 0
      max_surge                     = "10%"
      node_soak_duration_in_minutes = 0
    }
  }

  identity {
    type = "SystemAssigned"
  }

  network_profile {
    network_plugin = "azure"
    network_policy = "azure"
    outbound_type  = "userDefinedRouting"
  }

  depends_on = [
    azurerm_route_table.route_table,
    azurerm_subnet_route_table_association.route_table_association
  ]

  lifecycle {
    ignore_changes = [
      tags["CreatedAt"]
    ]
  }
}
