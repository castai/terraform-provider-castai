terraform {
  required_version = ">= 1.5" # check blocks require Terraform 1.5 or later

  required_providers {
    castai = {
      source = "castai/castai"
    }
    google = {
      source = "hashicorp/google"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.0"
    }
  }
}
