locals {
  role_name = "castai-eks-role"
}

provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.eks.endpoint
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks.certificate_authority[0].data)
    token                  = data.aws_eks_cluster_auth.eks.token
  }
}

provider "aws" {
  region = var.cluster_region
}

provider "castai" {
  api_token = var.castai_api_token
  api_url   = var.castai_api_url
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.eks.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.eks.token
}

data "aws_caller_identity" "current" {}

data "castai_eks_clusterid" "castai_cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}

data "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = data.castai_eks_clusterid.castai_cluster_id.id
}

module "castai-eks-cluster" {
  source = "castai/eks-cluster/castai"

  api_url             = var.castai_api_url
  aws_account_id      = data.aws_caller_identity.current.account_id
  aws_cluster_region  = var.cluster_region
  aws_cluster_name    = module.eks.cluster_id
  aws_assume_role_arn = aws_iam_role.assume_role.arn

  default_node_configuration = module.castai-eks-cluster.castai_node_configurations["default"]

  node_configurations = {
    default = {
      subnets         = module.vpc.private_subnets
      tags            = var.tags
      security_groups = [
        module.eks.cluster_security_group_id,
        module.eks.node_security_group_id,
        aws_security_group.additional.id,
      ]
      instance_profile_arn = aws_iam_instance_profile.castai_instance_profile.arn
    }
  }

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
}