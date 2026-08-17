locals {
  name = var.cluster_name
  tags = var.tags
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 21.0"

  name               = local.name
  kubernetes_version = var.cluster_version

  enable_cluster_creator_admin_permissions = true
  endpoint_public_access                   = true

  compute_config = {
    enabled = true
    # Built-in Auto Mode node pools, both EKS-managed. A self-managed
    # karpenter.sh NodePool would allow spot capacity, but managing it via
    # kubernetes_manifest needs a live cluster at plan time, forcing a
    # two-phase apply.
    node_pools = ["system", "general-purpose"]
  }

  addons = {
    coredns = {}
    eks-pod-identity-agent = {
      before_compute = true
    }
    kube-proxy = {
      before_compute = true
    }
    vpc-cni = {
      before_compute = true
    }
  }

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  tags = local.tags
}
