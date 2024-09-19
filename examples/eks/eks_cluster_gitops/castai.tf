# Create IAM resources required for connecting cluster to CAST AI.
locals {
  resource_name_postfix = var.aws_cluster_name
  account_id            = data.aws_caller_identity.current.account_id
  partition             = data.aws_partition.current.partition

  instance_profile_role_name = "castai-eks-${local.resource_name_postfix}-node-role"
  iam_role_name              = "castai-eks-${local.resource_name_postfix}-cluster-role"
  iam_inline_policy_name     = "CastEKSRestrictedAccess"
  role_name                  = "castai-eks-role"
}

data "aws_caller_identity" "current" {}

data "aws_partition" "current" {}

data "aws_eks_cluster" "existing_cluster" {
  name = var.aws_cluster_name
}

# Configure EKS cluster connection using CAST AI eks-cluster module.
resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.aws_cluster_region
  cluster_name = var.aws_cluster_name
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

module "castai-eks-role-iam" {
  source = "castai/eks-role-iam/castai"

  aws_account_id     = data.aws_caller_identity.current.account_id
  aws_cluster_region = var.aws_cluster_region
  aws_cluster_name   = var.aws_cluster_name
  aws_cluster_vpc_id = var.vpc_id

  castai_user_arn = castai_eks_user_arn.castai_user_arn.arn

  create_iam_resources_per_cluster = true
}

# Creates access entry if eks auth mode is API/API_CONFIGMAP
locals {
  access_entry = can(regex("API", data.aws_eks_cluster.existing_cluster.access_config[0].authentication_mode))
}

resource "aws_eks_access_entry" "access_entry" {
  count         = local.access_entry ? 1 : 0
  cluster_name  = local.resource_name_postfix
  principal_arn = module.castai-eks-role-iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}

# Connect eks cluster to CAST AI
resource "castai_eks_cluster" "my_castai_cluster" {
  account_id                 = var.aws_account_id
  region                     = var.aws_cluster_region
  name                       = local.resource_name_postfix
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
  assume_role_arn            = module.castai-eks-role-iam.role_arn
}

# Creates node configuration
resource "castai_node_configuration" "default" {
  cluster_id     = castai_eks_cluster.my_castai_cluster.id
  name           = "default"
  disk_cpu_ratio = 0
  min_disk_size  = 100
  subnets        = var.subnets
  eks {
    security_groups = [
      var.cluster_security_group_id,
      var.node_security_group_id
    ]
    instance_profile_arn = module.castai-eks-role-iam.instance_profile_arn
  }
}


# Promotes node configuration as default node configuration
resource "castai_node_configuration_default" "this" {
  cluster_id       = castai_eks_cluster.my_castai_cluster.id
  configuration_id = castai_node_configuration.default.id
}

resource "castai_node_template" "default_by_castai" {
  cluster_id = castai_eks_cluster.my_castai_cluster.id

  name             = "default-by-castai"
  is_default       = true
  is_enabled       = true
  configuration_id = castai_node_configuration.default.id
  should_taint     = true

  custom_labels = {
    env = "production"
  }

  constraints {
    on_demand = true
  }

}

resource "castai_node_template" "example_spot_template" {
  cluster_id = castai_eks_cluster.my_castai_cluster.id

  name             = "example_spot_template"
  is_default       = true
  is_enabled       = true
  configuration_id = castai_node_configuration.default.id
  should_taint     = true

  custom_labels = {
    env = "production"
  }

  custom_taints {
    key    = "dedicated"
    value  = "backend"
    effect = "NoSchedule"
  }

  constraints {
    on_demand                                   = true
    spot                                        = false
    use_spot_fallbacks                          = true
    fallback_restore_rate_seconds               = 300
    enable_spot_diversity                       = true
    spot_diversity_price_increase_limit_percent = 20
    spot_interruption_predictions_enabled       = true
    spot_interruption_predictions_type          = "aws-rebalance-recommendations"
    compute_optimized_state                     = "disabled"
    storage_optimized_state                     = "disabled"
    is_gpu_only                                 = false
    min_cpu                                     = 2
    max_cpu                                     = 8
    min_memory                                  = 4096
    max_memory                                  = 16384
    architectures                               = ["amd64"]
    azs                                         = ["eu-central-1a", "eu-central-1b"]
    burstable_instances                         = "disabled"
    customer_specific                           = "disabled"

    instance_families {
      include = ["c5"]
    }

    custom_priority {
      instance_families = ["c5"]
      spot              = false
      on_demand         = true
    }
  }

}

resource "castai_autoscaler" "castai_autoscaler_policy" {
  cluster_id = castai_eks_cluster.my_castai_cluster.id

  autoscaler_settings {
    enabled                                 = true
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods {
      enabled = true
    }

    cluster_limits {
      enabled = true

      cpu {
        min_cores = 1
        max_cores = 10
      }
    }

    node_downscaler {
      enabled = true

      empty_nodes {
        enabled       = true
        delay_seconds = 90
      }

      evictor {
        enabled                                = true
        dry_run                                = false
        aggressive_mode                        = false
        scoped_mode                            = false
        cycle_interval                         = "60s"
        node_grace_period_minutes              = 10
        pod_eviction_failure_back_off_interval = "30s"
        ignore_pod_disruption_budgets          = false
      }
    }
  }
}
