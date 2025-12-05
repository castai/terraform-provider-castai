terraform {
  required_version = ">= 1.3.2"

  required_providers {
    castai = {
      source  = "castai/castai"
      version = ">= 3.11.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}