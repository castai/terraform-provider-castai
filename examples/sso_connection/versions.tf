terraform {
  required_providers {
    castai = {
      source  = "castai/castai"
      version = ">= 5.8.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.79.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = ">= 2.45.0"
    }
  }
}
