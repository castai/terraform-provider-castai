# 1. read virtual network and resource group for the cluster.

data "azurerm_subnet" "internal" {
  name                 = var.subnet_name
  virtual_network_name = var.vnet_name
  resource_group_name  = var.subnet_resource_group_name
}