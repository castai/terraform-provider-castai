# 2. Create EKS cluster.
module "eks" {
  source       = "terraform-aws-modules/eks/aws"
  version      = "19.4.2"
  putin_khuylo = true

  cluster_name                   = var.cluster_name
  cluster_version                = var.cluster_version
  cluster_endpoint_public_access = true

  cluster_addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent = true
      configuration_values = jsonencode({
        env = {
          # Enable prefix delegation for better IP efficiency
          ENABLE_PREFIX_DELEGATION = "true"
          # Allocate 1 warm prefix per node (/28 = 16 IPs)
          WARM_PREFIX_TARGET = "1"
          # Disable warm IP pool since we're using prefix mode
          WARM_IP_TARGET = "0"
          # Minimum IPs to maintain (in prefix mode, this is less critical)
          MINIMUM_IP_TARGET = "2"
        }
      })
    }
  }

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  manage_aws_auth_configmap = true

  aws_auth_roles = [
    # Add the CAST AI IAM role which required for CAST AI nodes to join the cluster.
    {
      rolearn  = module.castai-eks-role-iam.instance_profile_role_arn
      username = "system:node:{{EC2PrivateDNSName}}"
      groups = [
        "system:bootstrappers",
        "system:nodes",
      ]
    },
  ]

  eks_managed_node_groups = {
    node_group_spot = {
      name         = "${var.cluster_name}-spot"
      min_size     = 3
      max_size     = 10
      desired_size = 4

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
      "100.64.0.0/16", # Match VPC CIDR range
    ]
  }
}
