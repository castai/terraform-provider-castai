# Basics - resource group and VNet for the Azure resources.

resource "azurerm_resource_group" "this" {
  name     = local.name
  location = var.cluster_region
  tags     = local.tags
}

resource "azurerm_virtual_network" "this" {
  name                = "${local.name}-vnet"
  location            = azurerm_resource_group.this.location
  resource_group_name = azurerm_resource_group.this.name
  address_space       = [local.vnet_address_space]
  tags                = local.tags
}

# Node subnet of the AKS cluster.
resource "azurerm_subnet" "aks_nodes" {
  name                 = "aks-nodes"
  resource_group_name  = azurerm_resource_group.this.name
  virtual_network_name = azurerm_virtual_network.this.name
  address_prefixes     = [local.aks_subnet_prefix]
}

# Subnet that hosts the CAST AI private endpoint. Kept separate from the nodes
# because a private endpoint subnet cannot enforce network policies (NSGs/UDRs);
# that restriction is why `private_endpoint_network_policies` is Disabled.
resource "azurerm_subnet" "private_endpoints" {
  name                              = "private-endpoints"
  resource_group_name               = azurerm_resource_group.this.name
  virtual_network_name              = azurerm_virtual_network.this.name
  address_prefixes                  = [local.pe_subnet_prefix]
  private_endpoint_network_policies = "Disabled"
}
