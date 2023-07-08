terraform {
  required_version = ">= 0.13"

  required_providers {
    castai = {
      source  = "castai/castai"
      version = ">= 3.11.0"
    }
  }
}