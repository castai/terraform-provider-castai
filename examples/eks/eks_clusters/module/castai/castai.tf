locals {
  role_name = "castai-eks-role"

  default_node_cfg = {
    default = {
      subnets              = var.subnets
      tags                 = var.tags
      security_groups      = var.security_groups
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
    }
  }

  default_node_tmpl = {
    default_by_castai = {
      name = "default-by-castai"
      configuration_id = module.castai-eks-cluster.castai_node_configurations[
        "default"
      ]
      is_default   = true
      is_enabled   = true
      should_taint = false

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
  }

  node_configuration = merge(local.default_node_cfg, var.only_default == true ? {} : {
    test_node_config = {
      subnets              = var.subnets
      tags                 = var.tags
      security_groups      = var.security_groups
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
      kubelet_config = jsonencode({
        "registryBurst" : 20,
        "registryPullQPS" : 10
      })
      container_runtime = "containerd"
      volume_type       = "gp3"
      volume_iops       = 3100
      volume_throughput = 130
      imds_v1           = true
    }
  })

  node_templates = merge(local.default_node_tmpl, var.only_default == true ? {} : {
    spot_tmpl = {
      configuration_id = module.castai-eks-cluster.castai_node_configurations["default"]
      is_enabled       = true
      should_taint     = true

      custom_labels = {
        custom-label-key-1 = "custom-label-value-1"
        custom-label-key-2 = "custom-label-value-2"
      }

      custom_taints = [
        {
          key   = "custom-taint-key-1"
          value = "custom-taint-value-1"
        },
        {
          key   = "custom-taint-key-2"
          value = "custom-taint-value-2"
        }
      ]

      constraints = {
        fallback_restore_rate_seconds = 1800
        spot                          = true
        use_spot_fallbacks            = true
        min_cpu                       = 4
        max_cpu                       = 100
        instance_families = {
          exclude = ["m5"]
        }
        compute_optimized_state = "disabled"
        storage_optimized_state = "disabled"
        is_gpu_only             = false
      }
    }
  })
}

# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

# Create AWS IAM policies and a user to connect to CAST AI.
module "castai-eks-role-iam" {
  source  = "castai/eks-role-iam/castai"
  version = "~> 1.0"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = var.vpc_id

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
  version = "~> 13.0"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name

  aws_assume_role_arn        = module.castai-eks-role-iam.role_arn
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration = module.castai-eks-cluster.castai_node_configurations["default"]

  node_configurations = local.node_configuration

  node_templates = local.default_node_tmpl

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
  depends_on = [module.castai-eks-role-iam]
}

