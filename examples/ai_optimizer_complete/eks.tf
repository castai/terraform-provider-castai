# EKS Cluster.
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 21.0"

  name                   = var.cluster_name
  kubernetes_version     = var.cluster_version
  endpoint_public_access = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  enable_cluster_creator_admin_permissions = true

  # Core addons.
  addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent    = true
      before_compute = true
    }
    vpc-cni = {
      most_recent    = true
      before_compute = true
    }
  }

  # GPU node group for AI Optimizer.
  eks_managed_node_groups = {
    ai_optimizer_gpu = {
      name         = "${var.cluster_name}-ai-gpu"
      min_size     = 1
      max_size     = 4
      desired_size = 1

      instance_types = ["g5.xlarge"]
      capacity_type  = "SPOT"

      labels = {
        "cast.ai/workload" = "ai-optimizer"
        "nvidia.com/gpu.present" = "true"
      }

      taints = [{
        key    = "nvidia.com/gpu"
        value  = "true"
        effect = "NO_SCHEDULE"
      }]

      metadata_options = {
        http_endpoint               = "enabled"
        http_tokens                 = "required"
        http_put_response_hop_limit = 2
      }
    }

    # General purpose node group.
    general = {
      name         = "${var.cluster_name}-general"
      min_size     = 2
      max_size     = 5
      desired_size = 2

      instance_types = ["t3.large"]
      capacity_type  = "SPOT"

      metadata_options = {
        http_endpoint               = "enabled"
        http_tokens                 = "required"
        http_put_response_hop_limit = 2
      }
    }
  }
}

# CAST AI access entry for nodes.
resource "aws_eks_access_entry" "castai" {
  cluster_name  = module.eks.cluster_name
  principal_arn = module.castai-eks-role-iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}
