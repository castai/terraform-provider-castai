module "cluster" {
  source = "./module/castai"
  count  = var.enable_castai ? 1 : 0

  cluster_name     = var.cluster_name
  castai_api_token = var.castai_api_token
  cluster_region   = var.cluster_region
  vpc_id           = module.vpc.vpc_id
  security_groups = [
    module.eks.cluster_security_group_id,
    module.eks.node_security_group_id,
    aws_security_group.additional.id,
  ]
  subnets = module.vpc.private_subnets
}
