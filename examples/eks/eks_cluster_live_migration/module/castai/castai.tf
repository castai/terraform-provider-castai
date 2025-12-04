locals {
  default_node_cfg = {
    default = {
      subnets              = var.subnets
      tags                 = var.tags
      security_groups      = var.security_groups
      instance_profile_arn = var.castai-eks-role-iam_instance_profile_arn
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

  node_configuration = merge(local.default_node_cfg, {
    live = {
      subnets              = var.subnets,
      instance_profile_arn = var.castai-eks-role-iam_instance_profile_arn
      security_groups      = var.security_groups

      container_runtime = "containerd"
      eks_image_family  = "al2023"
    }
  })

  node_templates = merge(local.default_node_tmpl, {
    live_tmpl = {
      configuration_id = module.castai-eks-cluster.castai_node_configurations["live"]
      is_enabled       = true
      should_taint     = true
      clm_enabled      = true
    }
  })
}

# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

module "castai-eks-cluster" {
  source  = "castai/eks-cluster/castai"
  version = "13.6.2"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name

  aws_assume_role_arn        = var.castai-eks-role-iam_role_arn
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration = module.castai-eks-cluster.castai_node_configurations["default"]

  node_configurations = local.node_configuration

  node_templates = local.node_templates

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

  install_live = var.install_helm_live
  live_version = var.install_helm_live ? var.live_helm_version : null
}
