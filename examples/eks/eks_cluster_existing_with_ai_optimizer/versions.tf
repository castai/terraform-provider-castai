terraform {
  required_version = ">= 1.11"

  required_providers {
    castai = {
      source = "castai/castai"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = ">= 6.23.0"
    }
  }
}
