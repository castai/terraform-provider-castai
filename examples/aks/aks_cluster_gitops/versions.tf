terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
    }
    azuread = {
      source = "hashicorp/azuread"
    }
    castai = {
      source = "castai/castai"
    }
  }
  required_version = ">= 0.13"
}
