provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}
provider "aws" {
  region  = var.aws_cluster_region
  profile = var.profile
}
