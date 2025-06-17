terraform {
  required_version = ">= 0.13"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.95.0"
    }
    castai = {
      source  = "castai/castai"
      version = "7.51.0"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
  }
}