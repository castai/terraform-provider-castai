# Following providers required by AKS and Vnet resources.
provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
}

provider "azuread" {
  tenant_id = data.azurerm_subscription.current.tenant_id
}
