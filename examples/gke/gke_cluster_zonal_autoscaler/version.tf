terraform {
  required_providers {
    castai = {
      source = "castai/castai"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
    google = {
      source = "hashicorp/google"
    }
    google-beta = {
      source = "hashicorp/google-beta"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
    }
  }
  required_version = ">= 0.13"
}