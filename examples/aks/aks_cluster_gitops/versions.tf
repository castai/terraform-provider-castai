terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 5.0"
    }
    azuread = {
      source = "hashicorp/azuread"
    }
    castai = {
      source = "castai/castai"
    }
  }
  required_version = ">= 1.3.2"
}
