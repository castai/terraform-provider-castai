module "cluster" {
  source = "./module/castai"
  count  = var.enable_castai ? 1 : 0

  cluster_name     = var.cluster_name
  cluster_region   = var.cluster_region
  castai_api_token = var.castai_api_token
  castai_api_url   = var.castai_api_url
  castai_grpc_url  = var.castai_grpc_url

  vpc_id = module.vpc.vpc_id
  security_groups = [
    module.eks.cluster_security_group_id,
    module.eks.node_security_group_id,
    aws_security_group.additional.id,
  ]
  subnets            = module.vpc.private_subnets
  live_proxy_version = var.live_proxy_version
  live_helm_version  = var.live_helm_version

  install_helm_live = var.install_helm_live

  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
}
