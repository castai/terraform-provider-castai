# 2. Create EKS cluster.

module "eks" {
  source       = "terraform-aws-modules/eks/aws"
  version      = "21.10.1"
  putin_khuylo = true

  name                   = var.cluster_name
  kubernetes_version     = var.cluster_version
  endpoint_public_access = true

  addons = {
    coredns    = {}
    kube-proxy = {}
    vpc-cni    = {}
  }

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  enable_cluster_creator_admin_permissions = true
  authentication_mode                      = "API_AND_CONFIG_MAP"

  eks_managed_node_groups = {
    node_group_1 = {
      name           = "${var.cluster_name}-ng-1"
      instance_types = ["m5.large", "m5.xlarge", "t3.large"]
      desired_size   = 2
    }
  }
}

# Add the CAST AI IAM role which is required for CAST AI nodes to join the cluster.
resource "aws_eks_access_entry" "castai" {
  cluster_name  = module.eks.cluster_name
  principal_arn = module.castai-eks-role-iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}
