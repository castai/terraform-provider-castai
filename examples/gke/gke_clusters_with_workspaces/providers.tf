provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes {
    config_path    = "~/.kube/config"
    config_context = var.kube_config_context
  }
}
