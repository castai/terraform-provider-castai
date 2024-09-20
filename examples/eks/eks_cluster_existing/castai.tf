# Configure Data sources and providers required for CAST AI connection.
data "aws_caller_identity" "current" {}

data "aws_eks_cluster" "existing_cluster" {
  name = var.cluster_name # Replace with the actual name of your EKS cluster
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

locals {
  access_entry = can(regex("API", data.aws_eks_cluster.existing_cluster.access_config[0].authentication_mode))
}

resource "aws_eks_access_entry" "access_entry" {
  count         = local.access_entry ? 1 : 0
  cluster_name  = var.cluster_name
  principal_arn = module.castai-eks-role-iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}

# Create AWS IAM policies and a user to connect to CAST AI.
module "castai-eks-role-iam" {
  source = "castai/eks-role-iam/castai"

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
  source = "castai/eks-cluster/castai"

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
  node_configurations = {
    default = {
      subnets = var.subnets
      tags    = var.tags
      security_groups = [
        var.cluster_security_group_id,
        var.node_security_group_id
      ]
      instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
    }
  }


  node_templates = {
    default_by_castai = {
      name             = "default-by-castai"
      configuration_id = module.castai-eks-cluster.castai_node_configurations["default"]
      is_default       = true
      is_enabled       = true
      should_taint     = false

      constraints = {
        on_demand          = true
        spot               = false
        use_spot_fallbacks = false

        enable_spot_diversity                       = false
        spot_diversity_price_increase_limit_percent = 20

        spot_interruption_predictions_enabled = false
        spot_interruption_predictions_type    = "aws-rebalance-recommendations"
      }
    }
    example_spot_template = {
      configuration_id = module.castai-eks-cluster.castai_node_configurations["default"]
      is_enabled       = true
      should_taint     = true

      custom_labels = {
        custom-label-key-1 = "custom-label-value-1",
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
        is_gpu_only = false

        # Optional: define custom priority for instances selection.
        #
        # 1. Prioritize C5a and C5ad spot instances above all else, regardless of price.
        # 2. If C5a is not available, try C6a family.
        custom_priority = [
          {
            instance_families = ["c5a", "c5ad"]
            spot              = true
          },
          {
            instance_families = ["c6a"]
            spot              = true
          }
          # 3. instances not matching any of custom priority groups will be tried after
          # nothing matches from priority groups.
        ]
      }
    }
  }
  # Autoscaling & evictor setting
  autoscaler_settings = {
    enabled                                 = false
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods = {
      enabled = false
    }

    node_downscaler = {
      enabled = false

      empty_nodes = {
        enabled = false
      }

      evictor = {
        enabled                   = false
        aggressive_mode           = false
        cycle_interval            = "60s"
        node_grace_period_minutes = 10
        scoped_mode               = false
      }
    }

    cluster_limits = {
      enabled = false

      cpu = {
        max_cores = 20
        min_cores = 1
      }

      spot_backups = {
        enabled                          = false
        spot_backup_restore_rate_seconds = 1800
      }
    }
  }
  # Installs Workload autoscaler
  install_workload_autoscaler = true
  # Installs network monitor
  install_egressd = true

  # depends_on helps Terraform with creating proper dependencies graph in case of resource creation and in this case destroy.
  # module "castai-eks-cluster" has to be destroyed before module "castai-eks-role-iam".
  depends_on = [module.castai-eks-role-iam]
}

resource "castai_rebalancing_schedule" "spots" {
  name = "rebalance spots at every 30th minute"
  schedule {
    cron = "*/30 * * * *"
  }
  trigger_conditions {
    savings_percentage = 20
  }
  launch_configuration {
    # only consider instances older than 5 minutes
    node_ttl_seconds         = 300
    num_targeted_nodes       = 3
    rebalancing_min_nodes    = 2
    keep_drain_timeout_nodes = false
    selector = jsonencode({
      nodeSelectorTerms = [{
        matchExpressions = [
          {
            key      = "scheduling.cast.ai/spot"
            operator = "Exists"
          }
        ]
      }]
    })
    execution_conditions {
      enabled                     = true
      achieved_savings_percentage = 10
    }
  }
}

resource "castai_rebalancing_job" "spots" {
  cluster_id              = castai_eks_clusterid.cluster_id.id
  rebalancing_schedule_id = castai_rebalancing_schedule.spots.id
  enabled                 = true
  depends_on              = [module.castai-eks-cluster]
}
