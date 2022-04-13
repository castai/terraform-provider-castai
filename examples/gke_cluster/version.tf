terraform {
  required_providers {
    castai = {
      source = "castai/castai"
    }
    google = {
      source  = "hashicorp/google"
      version = "4.16.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "4.16.0"
    }
  }
  required_version = ">= 0.13"
}
