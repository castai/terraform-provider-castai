data "aws_caller_identity" "current" {}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}

module "castai-eks-role-iam" {
  source = "castai/eks-role-iam/castai"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  castai_user_arn = castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

resource "kubernetes_config_map" "castai_evictor_config" {
  metadata {
    name   = "castai-evictor-config"
    labels = {
      "app.kubernetes.io/name" = "castai-evictor-config"
    }
  }
  data = var.evictor_advanced_config
}