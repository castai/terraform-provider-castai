# 3. Connect EKS cluster to CAST AI.

locals {
  role_name = "castai-eks-role"
}

# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

provider "helm" {
  kubernetes = {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
    exec = {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      # This requires the awscli to be installed locally where Terraform is executed.
      args = ["eks", "get-token", "--cluster-name", module.eks.cluster_name, "--region", var.cluster_region]
    }
  }
}

# Create AWS IAM policies and a user to connect to CAST AI.
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

# Configure EKS cluster connection using CAST AI eks-cluster module.
resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}

module "castai-eks-cluster" {
  source  = "castai/eks-cluster/castai"
  version = "~> 14.1"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = module.eks.cluster_name

  aws_assume_role_arn        = module.castai-eks-role-iam.role_arn
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration_name = "default"

  node_configurations = {
    default = {
      subnets = module.vpc.private_subnets
      tags    = var.tags
      security_groups = [
        module.eks.cluster_security_group_id,
        module.eks.node_security_group_id,
        aws_security_group.additional.id,
      ]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
    }

    gpu = {
      subnets = module.vpc.private_subnets
      tags    = var.tags
      security_groups = [
        module.eks.cluster_security_group_id,
        module.eks.node_security_group_id,
        aws_security_group.additional.id,
      ]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
      container_runtime    = "containerd"
      volume_type          = "gp3"
      volume_iops          = 3000
      volume_throughput    = 125
    }
  }

  node_templates = {
    default_by_castai = {
      name               = "default-by-castai"
      configuration_name = "default"
      is_default         = true
      is_enabled         = true
      should_taint       = false

      constraints = {
        on_demand          = true
        spot               = true
        use_spot_fallbacks = true

        enable_spot_diversity                       = false
        spot_diversity_price_increase_limit_percent = 20

        spot_interruption_predictions_enabled = true
        spot_interruption_predictions_type    = "aws-rebalance-recommendations"
      }
    }

    gpu = {
      name               = "gpu"
      configuration_name = "gpu"
      is_enabled         = true
      should_taint       = true

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

  autoscaler_settings = {
    enabled                                 = true
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods = {
      enabled = true
    }

    node_downscaler = {
      enabled = true

      empty_nodes = {
        enabled = true
      }

      evictor = {
        aggressive_mode           = false
        cycle_interval            = "5m10s"
        dry_run                   = false
        enabled                   = true
        node_grace_period_minutes = 10
        scoped_mode               = false
      }
    }

    cluster_limits = {
      enabled = true

      cpu = {
        max_cores = 20
        min_cores = 1
      }
    }
  }

  # depends_on helps Terraform with creating proper dependencies graph in case of resource creation and in this case destroy.
  # module "castai-eks-cluster" has to be destroyed before module "castai-eks-role-iam".
  depends_on = [module.castai-eks-role-iam, module.vpc, module.eks]
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

  set = [
    {
      name  = "castai.apiKey"
      value = var.castai_api_token
    },
    {
      name  = "castai.clusterID"
      value = castai_eks_clusterid.cluster_id.id
    }
  ]

  depends_on = [module.castai-eks-cluster]
}
