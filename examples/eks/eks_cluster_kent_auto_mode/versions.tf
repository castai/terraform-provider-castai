terraform {
  required_providers {
    castai = {
      source  = "castai/castai"
      version = "~> 7.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
  required_version = ">= 1.3.2"
}
