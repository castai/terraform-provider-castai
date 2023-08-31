provider "castai" {
  alias     = "dev"
  api_token = var.castai_dev_api_token
}

provider "castai" {
  alias     = "prod"
  api_token = var.castai_prod_api_token
}