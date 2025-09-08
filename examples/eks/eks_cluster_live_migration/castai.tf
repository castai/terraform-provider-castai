# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

# Configure EKS cluster connection using CAST AI eks-cluster module.
resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

# Create AWS IAM policies and a user to connect to CAST AI.
module "castai-eks-role-iam" {
  source  = "castai/eks-role-iam/castai"
  version = "~> 1.0"
  count  = var.enable_castai ? 1 : 0

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  castai_user_arn = castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

module "cluster" {
  source = "./module/castai"
  count  = var.enable_castai ? 1 : 0

  cluster_name     = var.cluster_name
  cluster_region   = var.cluster_region
  castai_api_token = var.castai_api_token
  castai_api_url   = var.castai_api_url
  castai_grpc_url  = var.castai_grpc_url

  castai-eks-role-iam_instance_profile_arn = module.castai-eks-role-iam[0].instance_profile_role_arn
  castai-eks-role-iam_role_arn             = module.castai-eks-role-iam[0].role_arn

  security_groups = [
    module.eks.cluster_security_group_id,
    module.eks.node_security_group_id,
    aws_security_group.additional.id,
  ]
  subnets           = module.vpc.private_subnets
  live_helm_version = var.live_helm_version

  install_helm_live = var.install_helm_live

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  depends_on = [module.eks, module.castai-eks-role-iam]
}
