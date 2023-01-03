# Following providers required by AKS and Vnet resources.
provider "azurerm" {
  features {}
}

provider "azuread" {
  tenant_id = data.azurerm_subscription.current.tenant_id
}
