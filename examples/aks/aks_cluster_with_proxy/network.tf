resource "azurerm_virtual_network" "vnet" {
  name                = "aks-vnet"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  address_space       = ["10.42.0.0/16"]
}

resource "azurerm_subnet" "aks_subnet" {
  name                 = "aks-subnet"
  resource_group_name  = azurerm_resource_group.rg.name
  virtual_network_name = azurerm_virtual_network.vnet.name
  address_prefixes     = ["10.42.1.0/24"]
}

resource "azurerm_subnet" "egress_firewall_subnet" {
  name                 = "AzureFirewallSubnet"
  resource_group_name  = azurerm_resource_group.rg.name
  virtual_network_name = azurerm_virtual_network.vnet.name
  address_prefixes     = ["10.42.2.0/24"]
}


resource "azurerm_virtual_network" "proxy_vnet" {
  name                = "proxy-vnet"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  address_space       = ["10.43.0.0/16"]
  lifecycle {
    ignore_changes = [tags]
  }
}

resource "azurerm_subnet" "explicit_firewall_subnet" {
  name                 = "AzureFirewallSubnet"
  resource_group_name  = azurerm_resource_group.rg.name
  virtual_network_name = azurerm_virtual_network.proxy_vnet.name
  address_prefixes     = ["10.43.1.0/24"]
}

resource "azurerm_virtual_network_peering" "peer_vnets" {
  name                         = "peer-firewall-vnets"
  resource_group_name          = azurerm_resource_group.rg.name
  virtual_network_name         = azurerm_virtual_network.proxy_vnet.name
  remote_virtual_network_id    = azurerm_virtual_network.vnet.id
  allow_virtual_network_access = true

  depends_on = [azurerm_virtual_network.proxy_vnet]
}


resource "azurerm_virtual_network_peering" "peer_vnets_reverse" {
  name                         = "peer-firewall-vnets-reverse"
  resource_group_name          = azurerm_resource_group.rg.name
  virtual_network_name         = azurerm_virtual_network.vnet.name
  remote_virtual_network_id    = azurerm_virtual_network.proxy_vnet.id
  allow_virtual_network_access = true

  depends_on = [azurerm_virtual_network.proxy_vnet]
}

