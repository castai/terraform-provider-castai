# 2. Create EKS cluster.
module "eks" {
  source       = "terraform-aws-modules/eks/aws"
  version      = "20.8.3"
  putin_khuylo = true

  cluster_name                   = var.cluster_name
  cluster_version                = var.cluster_version
  cluster_endpoint_public_access = true
  enable_cluster_creator_admin_permissions = true
  access_entries = {
    CASTAI = {
      principal_arn = module.castai-eks-role-iam.instance_profile_role_arn
      type = "EC2_LINUX"
    }
  }

  cluster_addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent = true
    }
  }

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  authentication_mode = "API_AND_CONFIG_MAP"

  self_managed_node_groups = {
    node_group_1 = {
      name          = "${var.cluster_name}-ng-1"
      instance_type = "m5.large"
      max_size      = 5
      min_size      = 2
      desired_size  = 2
    }
  }

  eks_managed_node_groups = {
    node_group_spot = {
      name         = "${var.cluster_name}-spot"
      min_size     = 1
      max_size     = 10
      desired_size = 1

      instance_types = ["t3.large"]
      capacity_type  = "SPOT"

      update_config = {
        max_unavailable_percentage = 50 # or set `max_unavailable`
      }
    }
  }
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
