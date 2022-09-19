# Your EKS cluster access configuration.

provider "aws" {
  region = var.cluster_region
}

data "aws_eks_cluster" "cluster" {
  name = var.cluster_name
}

data "aws_eks_cluster_auth" "cluster" {
  name = var.cluster_name
}

provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.cluster.endpoint
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority[0].data)
    token                  = data.aws_eks_cluster_auth.cluster.token
  }
}

data "aws_caller_identity" "current" {}

# Your CAST AI EKS configuration

provider "castai" {
  api_token = var.castai_api_token
}

resource "castai_cluster_token" "this" {
  cluster_id = castai_eks_cluster.this.id
}

resource "castai_eks_cluster" "this" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.cluster_region
  name       = var.cluster_name
}

resource "helm_release" "castai_agent" {
  name            = "castai-agent"
  repository      = "https://castai.github.io/helm-charts"
  chart           = "castai-agent"
  cleanup_on_fail = true

  set {
    name  = "provider"
    value = "eks"
  }
  set_sensitive {
    name  = "apiKey"
    value = castai_cluster_token.this.cluster_token
  }
}
