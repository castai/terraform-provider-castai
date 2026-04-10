data "aws_caller_identity" "current" {}

data "aws_eks_cluster" "existing_cluster" {
  name = var.cluster_name
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

locals {
  access_entry = can(regex("API", data.aws_eks_cluster.existing_cluster.access_config[0].authentication_mode))
}

resource "aws_eks_access_entry" "access_entry" {
  count         = local.access_entry ? 1 : 0
  cluster_name  = data.aws_eks_cluster.existing_cluster.name
  principal_arn = module.castai_eks_role_iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}

module "castai_eks_role_iam" {
  source  = "castai/eks-role-iam/castai"
  version = "~> 2.0"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name
  aws_cluster_vpc_id = data.aws_eks_cluster.existing_cluster.vpc_config[0].vpc_id

  castai_user_arn = castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.cluster_region
  cluster_name = var.cluster_name
}

module "castai_eks_cluster" {
  source  = "castai/eks-cluster/castai"
  version = "~> 14.1"

  api_url                = var.castai_api_url
  castai_api_token       = var.castai_api_token
  grpc_url               = var.castai_grpc_url
  wait_for_cluster_ready = true

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.cluster_region
  aws_cluster_name   = var.cluster_name

  aws_assume_role_arn        = module.castai_eks_role_iam.role_arn
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  default_node_configuration = module.castai_eks_cluster.castai_node_configurations["default"]
  node_configurations = {
    default = {
      subnets = data.aws_eks_cluster.existing_cluster.vpc_config[0].subnet_ids
      tags    = var.tags
      security_groups = concat(
        [data.aws_eks_cluster.existing_cluster.vpc_config[0].cluster_security_group_id],
        tolist(data.aws_eks_cluster.existing_cluster.vpc_config[0].security_group_ids)
      )
      instance_profile_arn = module.castai_eks_role_iam.instance_profile_arn
    }
  }

  node_templates = {
    default_by_castai = {
      name             = "default-by-castai"
      configuration_id = module.castai_eks_cluster.castai_node_configurations["default"]
      is_default       = true
      is_enabled       = true
      should_taint     = false

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

  install_ai_optimizer = var.enable_ai_optimizer

  depends_on = [module.castai_eks_role_iam]
}
