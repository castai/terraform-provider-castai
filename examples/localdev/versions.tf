terraform {
  required_providers {
    castai = {
      source = "castai/castai"
    }

  }
  required_version = ">= 0.13"
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}
