terraform {
  required_providers {
    castai = {
      source = "castai/castai"
    }
    google = {
      source  = "hashicorp/google"
      version = "4.22.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "4.22.0"
    }
  }
  required_version = ">= 1.0"

  backend "remote" {
    organization = "CastAI"

    workspaces {
      name = "e2e-tf-gke-tf-e2e-gke"
    }
  }
}
