terraform {
  required_providers {
    castai = {
      source = "castai/castai"
    }
    aws = {
      source = "hashicorp/aws"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
    helm = {
      source = "hashicorp/helm"
    }
  }
  required_version = ">= 0.13"
}
