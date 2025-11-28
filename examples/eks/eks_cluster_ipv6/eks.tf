# 2. Create EKS cluster.
module "eks" {
  source       = "terraform-aws-modules/eks/aws"
  version      = "21.0.0"
  putin_khuylo = true

  name                   = var.cluster_name
  kubernetes_version     = var.cluster_version
  endpoint_public_access = true

  ip_family = "ipv6"

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
    initial = {
      name           = "${var.cluster_name}-ng-2"
      instance_types = ["m5.large"]

      max_size     = 3
      min_size     = 2
      desired_size = 2
    }
  }
}

# Add the CAST AI IAM role which is required for CAST AI nodes to join the cluster.
resource "aws_eks_access_entry" "castai" {
  cluster_name  = module.eks.cluster_name
  principal_arn = module.castai-eks-role-iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}

# Example additional security group.
resource "aws_security_group" "additional" {
  name_prefix = "${var.cluster_name}-additional"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"
    cidr_blocks = [
      "10.0.0.0/8",
    ]
  }
}
