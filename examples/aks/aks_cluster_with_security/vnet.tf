# 1. Create virtual network and resource group for the cluster.

resource "azurerm_resource_group" "this" {
  name     = var.cluster_name
  location = var.cluster_region
}

resource "azurerm_virtual_network" "this" {
  name                = "${var.cluster_name}-network"
  location            = azurerm_resource_group.this.location
  resource_group_name = azurerm_resource_group.this.name
  address_space       = ["10.1.0.0/16"]
}

resource "azurerm_subnet" "internal" {
  name                 = "internal"
  virtual_network_name = azurerm_virtual_network.this.name
  resource_group_name  = azurerm_resource_group.this.name
  address_prefixes     = ["10.1.0.0/22"]
}
