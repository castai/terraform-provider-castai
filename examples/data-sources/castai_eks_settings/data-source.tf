data "castai_eks_settings" "current" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.cluster_region
  cluster    = var.cluster_name
  vpc        = var.vpc
}

# Optionally limit the roles CAST AI may pass to EC2/EKS (default: any role).
# Keep this list in sync with the instance profile roles used in Node Configurations.
data "castai_eks_settings" "restricted" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.cluster_region
  cluster    = var.cluster_name
  vpc        = var.vpc

  instance_profile_role_arns = [
    "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/castai-eks-node-role",
  ]
}
