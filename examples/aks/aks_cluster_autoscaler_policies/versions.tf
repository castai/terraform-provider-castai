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
      version = "7.20.1"
    }
  }
  required_version = ">= 0.13"
}
