module "eks" {
  source       = "terraform-aws-modules/eks/aws"
  version      = "18.21.0"
  putin_khuylo = true

  cluster_name    = var.cluster_name
  cluster_version = "1.22"

  cluster_endpoint_private_access = true
  cluster_endpoint_public_access  = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  # Extend cluster security group rules
  cluster_security_group_additional_rules = {
    egress_nodes_ephemeral_ports_tcp = {
      description                = "To node 1025-65535"
      protocol                   = "tcp"
      from_port                  = 1025
      to_port                    = 65535
      type                       = "egress"
      source_node_security_group = true
    }
  }

  # Self Managed Node Group(s)
  self_managed_node_group_defaults = {
    vpc_security_group_ids = [aws_security_group.additional.id]
  }

  self_managed_node_groups = {
    one = {
      name          = "${var.cluster_name}-default"
      instance_type = "t3.large"
      max_size      = 5
      min_size      = 2
      desired_size  = 2
    }
  }

  # EKS Managed Node Group(s)
  eks_managed_node_group_defaults = {
    ami_type = "AL2_x86_64"

    attach_cluster_primary_security_group = true
    vpc_security_group_ids                = [aws_security_group.additional.id]
  }

  eks_managed_node_groups = {
    spot = {
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

  manage_aws_auth_configmap = true

  aws_auth_node_iam_role_arns_non_windows = [
    # Needed for CAST nodes to join cluster
    aws_iam_role.castai_instance_profile_role.arn,
  ]
}

data "aws_eks_cluster" "eks" {
  name = module.eks.cluster_id
}

data "aws_eks_cluster_auth" "eks" {
  name = module.eks.cluster_id
}
