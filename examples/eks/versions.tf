terraform {
  required_providers {
    castai     = {
      source  = "castai/castai"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
  }
  required_version = ">= 0.13"
}
