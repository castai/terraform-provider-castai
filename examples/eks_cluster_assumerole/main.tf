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

provider "castai" {
  api_token = var.castai_api_token
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.eks.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.eks.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.eks.token
}

data "aws_eks_cluster_auth" "eks" {
  name = module.eks.cluster_id
}

data "castai_eks_clusterid" "castai_cluster_id" {
  account_id   = var.aws_account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}

data "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = data.castai_eks_clusterid.castai_cluster_id.id
}

module "castai-eks-role-iam" {
  source = "castai/eks-role-iam/castai"

  aws_account_id     = var.aws_account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  castai_user_arn = data.castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

module "castai-eks-cluster" {
  source = "castai/eks-cluster/castai"

  aws_account_id     = var.aws_account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = module.eks.cluster_id

  aws_assume_role_arn      = module.castai-eks-role-iam.role_arn
  aws_instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn

  subnets                  = var.subnets
  override_security_groups = var.security_groups
  tags                     = var.tags

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  depends_on = [module.castai-eks-role-iam]
}