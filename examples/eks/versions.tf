terraform {
  required_providers {
    castai     = {
      source  = "castai/castai"
      version = "0.0.0-local"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
  }
  required_version = ">= 0.13"
}
