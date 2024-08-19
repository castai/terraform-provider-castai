data "castai_eks_settings" "current" {
  account_id = data.aws_caller_identity.current.account_id
  region     = var.cluster_region
  cluster    = var.cluster_name
  vpc        = var.vpc
}
