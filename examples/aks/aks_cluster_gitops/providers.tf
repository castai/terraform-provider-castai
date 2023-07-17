provider "azurerm" {
  features {}
}

provider "azuread" {
  tenant_id = data.azurerm_subscription.current.tenant_id
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}
