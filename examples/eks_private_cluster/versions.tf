terraform {
  required_providers {
    castai     = {
      source = "castai/castai"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
    helm       = {
      source = "hashicorp/helm"
    }
  }
  required_version = ">= 0.13"
}
