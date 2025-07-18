terraform {
  required_version = ">= 0.13"

  required_providers {
    castai = {
      source  = "castai/castai"
      version = ">= 6.0.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
