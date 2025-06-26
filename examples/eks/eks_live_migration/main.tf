terraform {
  required_version = ">= 0.13"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.95.0"
    }
    castai = {
      source  = "castai/castai"
      version = "~> 7.55"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.15.0" # Force v2.x instead of v3.x
    }
  }
}