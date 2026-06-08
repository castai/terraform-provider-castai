terraform {
  required_version = ">= 1.0"

  required_providers {
    castai = {
      source  = "castai/castai"
      version = ">= 8.2.0"
    }
  }
}