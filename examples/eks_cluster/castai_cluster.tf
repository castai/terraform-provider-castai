# CAST.AI cluster and agent registration to your EKS cluster

locals {
  cluster_name = var.cluster_name
}

provider "castai" {
  api_token = var.castai_api_token
}

resource "castai_eks_cluster" "my_castai_cluster" {
  account_id = var.aws_account_id
  region     = var.cluster_region
  name       = local.cluster_name

  access_key_id        = aws_iam_access_key.castai.id
  secret_access_key    = aws_iam_access_key.castai.secret
  instance_profile_arn = aws_iam_instance_profile.instance_profile.arn

  depends_on = [module.eks]
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
    value = castai_eks_cluster.my_castai_cluster.cluster_token
  }
}
