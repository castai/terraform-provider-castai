data "aws_caller_identity" "current" {}

data "aws_eks_cluster" "eks" {
  name = var.cluster_name
}

resource "castai_eks_cluster" "this" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.aws_cluster_region
  name       = data.aws_eks_cluster.eks.id

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
  assume_role_arn            = module.castai-eks-role-iam.role_arn
}