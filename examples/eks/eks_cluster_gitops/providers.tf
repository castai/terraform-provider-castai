provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_key
}
provider "aws" {
  region  = var.cluster_region
  profile = var.profile
}
