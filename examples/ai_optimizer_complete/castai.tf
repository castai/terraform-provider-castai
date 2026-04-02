# CAST AI connection.
data "aws_caller_identity" "current" {}

resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = module.eks.cluster_name
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

# IAM role for CAST AI
module "castai-eks-role-iam" {
  source  = "castai/eks-role-iam/castai"
  version = "~> 2.0"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = module.vpc.vpc_id

  castai_user_arn = castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

# CAST AI EKS cluster connection.
module "castai-eks-cluster" {
  source  = "castai/eks-cluster/castai"
  version = "~> 16.0"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name

  aws_assume_role_arn        = module.castai-eks-role-iam.role_arn
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  node_configurations = {
    default = {
      subnets              = module.vpc.private_subnets
      security_groups      = [module.eks.cluster_security_group_id, module.eks.node_security_group_id]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
      volume_type          = "gp3"
      volume_iops          = 3000
      volume_throughput    = 125
    }
    gpu = {
      subnets              = module.vpc.private_subnets
      security_groups      = [module.eks.cluster_security_group_id, module.eks.node_security_group_id]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
      volume_type          = "gp3"
      volume_iops          = 3000
      volume_throughput    = 125
      container_runtime    = "containerd"
    }
  }

  node_templates = {
    default = {
      name       = "default"
      is_default = true
      is_enabled = true
    }
    gpu = {
      name       = "gpu"
      is_enabled = true
      should_taint = true
      custom_taints = [
        {
          key    = "nvidia.com/gpu"
          value  = "true"
          effect = "NoSchedule"
        }
      ]
      custom_labels = {
        "nvidia.com/gpu.present" = "true"
      }
      constraints = {
        instance_families = {
          include = ["g5", "p4", "p5"]
        }
        min_cpu = 4
        max_cpu = 96
        spot    = true
      }
    }
  }
}

# Install AI Optimizer Helm chart.
resource "helm_release" "ai_optimizer" {
  count = var.enable_ai_optimizer ? 1 : 0

  name       = "ai-optimizer"
  repository = "https://castai.github.io/helm-charts"
  chart      = "castai-ai-optimizer"
  version    = "0.1.0"
  namespace  = "castai-agent"

  create_namespace = true

  set {
    name  = "castai.apiKey"
    value = var.castai_api_token
  }

  set {
    name  = "castai.clusterID"
    value = castai_eks_clusterid.cluster_id.id
  }

  depends_on = [module.castai-eks-cluster]
}
