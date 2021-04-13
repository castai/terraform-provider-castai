terraform {
  required_version = ">= 0.12.18"

  required_providers {
    castai = {
      source  = "castai/castai"
      version = "0.4.0"
    }

    cloudflare = {
      source = "cloudflare/cloudflare"
      version = "~> 2.0"
    }
  }
}

provider "google" {
  version = "3.21.0"
  credentials = var.gcp_credentials
  region = var.gcp_region
}

provider "google-beta" {
  version = "3.21.0"
  credentials = var.gcp_credentials
  region = var.gcp_region
}

provider "template" {
  version = "~> 2.1.2"
}

provider "random" {
  version = "~> 2.2.1"
}

provider "castai" {
  api_token = var.castai_api_token
  api_url = var.castai_api_url
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

provider "kubernetes" {
  host = castai_cluster.cicd.kubeconfig.0.host
  cluster_ca_certificate = base64decode(castai_cluster.cicd.kubeconfig.0.cluster_ca_certificate)
  client_key = base64decode(castai_cluster.cicd.kubeconfig.0.client_key)
  client_certificate = base64decode(castai_cluster.cicd.kubeconfig.0.client_certificate)
}

provider "helm" {
  kubernetes {
    host = castai_cluster.cicd.kubeconfig.0.host
    cluster_ca_certificate = base64decode(castai_cluster.cicd.kubeconfig.0.cluster_ca_certificate)
    client_key = base64decode(castai_cluster.cicd.kubeconfig.0.client_key)
    client_certificate = base64decode(castai_cluster.cicd.kubeconfig.0.client_certificate)
  }
}
