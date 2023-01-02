terraform {
  required_version = ">= 0.13"

  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
      version = "3.7.0"
    }
    azuread = {
      source = "hashicorp/azuread"
      version = "2.22.0"
    }
    castai = {
      source  = "castai/castai"
    }
  }
}
