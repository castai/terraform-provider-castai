terraform {
  required_providers {
    castai = {
      source = "castai/castai"
      version = "0.0.0-local"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
    google = {
      source  = "hashicorp/google"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
    }
  }
  required_version = ">= 0.13"
}
