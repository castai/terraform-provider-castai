# Following providers required by AKS and Vnet resources.
provider "azurerm" {
  features {}
}

provider "azuread" {
  version   = "2.53.1"
  tenant_id = data.azurerm_subscription.current.tenant_id
}
