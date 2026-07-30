# Create IAM resources required for connecting cluster to CAST AI.
data "aws_caller_identity" "current" {}

data "aws_partition" "current" {}

data "aws_eks_cluster" "existing_cluster" {
  name = var.aws_cluster_name
}

resource "castai_eks_clusterid" "cluster_id" {
  account_id   = data.aws_caller_identity.current.account_id
  region       = var.aws_cluster_region
  cluster_name = var.aws_cluster_name
}

resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}

module "castai-eks-role-iam" {
  source  = "castai/eks-role-iam/castai"
  version = "~> 2.0"

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
  cluster_name  = data.aws_eks_cluster.existing_cluster.name
  principal_arn = module.castai-eks-role-iam.instance_profile_role_arn
  type          = "EC2_LINUX"
}

# Connect eks cluster to CAST AI
resource "castai_eks_cluster" "this" {
  account_id                 = var.aws_account_id
  region                     = var.aws_cluster_region
  name                       = var.aws_cluster_name
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
  assume_role_arn            = module.castai-eks-role-iam.role_arn
}

# Node configurations supplied by the caller (Terragrunt inputs), keyed by configuration name.
resource "castai_node_configuration" "this" {
  for_each = var.node_configurations

  cluster_id = castai_eks_cluster.this.id
  name       = each.key

  disk_cpu_ratio    = each.value.disk_cpu_ratio
  min_disk_size     = each.value.min_disk_size
  drain_timeout_sec = each.value.drain_timeout_sec
  subnets           = coalesce(each.value.subnets, var.subnets)
  ssh_public_key    = each.value.ssh_public_key
  image             = each.value.image
  init_script       = each.value.init_script
  container_runtime = each.value.container_runtime
  docker_config     = each.value.docker_config
  kubelet_config    = each.value.kubelet_config
  tags              = each.value.tags

  eks {
    security_groups      = coalesce(each.value.eks.security_groups, var.security_group_ids)
    instance_profile_arn = coalesce(each.value.eks.instance_profile_arn, module.castai-eks-role-iam.instance_profile_arn)

    node_group_arn                = each.value.eks.node_group_arn
    dns_cluster_ip                = each.value.eks.dns_cluster_ip
    key_pair_id                   = each.value.eks.key_pair_id
    volume_type                   = each.value.eks.volume_type
    volume_iops                   = each.value.eks.volume_iops
    volume_throughput             = each.value.eks.volume_throughput
    volume_kms_key_arn            = each.value.eks.volume_kms_key_arn
    imds_v1                       = each.value.eks.imds_v1
    imds_hop_limit                = each.value.eks.imds_hop_limit
    max_pods_per_node_formula     = each.value.eks.max_pods_per_node_formula
    ips_per_prefix                = each.value.eks.ips_per_prefix
    threads_per_cpu               = each.value.eks.threads_per_cpu
    eks_image_family              = each.value.eks.eks_image_family
    ena_queue_count_per_interface = each.value.eks.ena_queue_count_per_interface

    dynamic "target_group" {
      for_each = each.value.eks.target_group

      content {
        arn  = target_group.value.arn
        port = target_group.value.port
      }
    }
  }
}

# Promotes one of the node configurations as the cluster default.
resource "castai_node_configuration_default" "this" {
  cluster_id       = castai_eks_cluster.this.id
  configuration_id = castai_node_configuration.this[var.default_node_configuration].id

  lifecycle {
    precondition {
      condition     = contains(keys(var.node_configurations), var.default_node_configuration)
      error_message = "default_node_configuration must be a key of var.node_configurations."
    }
  }
}

# Node templates supplied by the caller (Terragrunt inputs), keyed by template name.
resource "castai_node_template" "this" {
  for_each = var.node_templates

  cluster_id = castai_eks_cluster.this.id

  name             = each.key
  is_default       = each.value.is_default
  is_enabled       = each.value.is_enabled
  should_taint     = each.value.should_taint
  configuration_id = castai_node_configuration.this[coalesce(each.value.node_configuration, var.default_node_configuration)].id

  custom_labels                = each.value.custom_labels
  custom_instances_enabled     = each.value.custom_instances_enabled
  rebalancing_config_min_nodes = each.value.rebalancing_config_min_nodes

  lifecycle {
    precondition {
      condition     = each.value.node_configuration == null || contains(keys(var.node_configurations), coalesce(each.value.node_configuration, var.default_node_configuration))
      error_message = "Node template '${each.key}' references node configuration '${coalesce(each.value.node_configuration, var.default_node_configuration)}' which is not defined in var.node_configurations."
    }
  }

  dynamic "custom_taints" {
    for_each = each.value.custom_taints

    content {
      key    = custom_taints.value.key
      value  = custom_taints.value.value
      effect = custom_taints.value.effect
    }
  }

  dynamic "constraints" {
    for_each = each.value.constraints == null ? [] : [each.value.constraints]

    content {
      on_demand                                   = constraints.value.on_demand
      spot                                        = constraints.value.spot
      use_spot_fallbacks                          = constraints.value.use_spot_fallbacks
      fallback_restore_rate_seconds               = constraints.value.fallback_restore_rate_seconds
      enable_spot_diversity                       = constraints.value.enable_spot_diversity
      spot_diversity_price_increase_limit_percent = constraints.value.spot_diversity_price_increase_limit_percent
      spot_interruption_predictions_enabled       = constraints.value.spot_interruption_predictions_enabled
      spot_interruption_predictions_type          = constraints.value.spot_interruption_predictions_type
      spot_reliability_enabled                    = constraints.value.spot_reliability_enabled
      compute_optimized_state                     = constraints.value.compute_optimized_state
      storage_optimized_state                     = constraints.value.storage_optimized_state
      is_gpu_only                                 = constraints.value.is_gpu_only
      min_cpu                                     = constraints.value.min_cpu
      max_cpu                                     = constraints.value.max_cpu
      min_memory                                  = constraints.value.min_memory
      max_memory                                  = constraints.value.max_memory
      architectures                               = constraints.value.architectures
      os                                          = constraints.value.os
      azs                                         = constraints.value.azs
      burstable_instances                         = constraints.value.burstable_instances
      customer_specific                           = constraints.value.customer_specific
      bare_metal                                  = constraints.value.bare_metal
      cpu_manufacturers                           = constraints.value.cpu_manufacturers

      dynamic "instance_families" {
        for_each = constraints.value.instance_families == null ? [] : [constraints.value.instance_families]

        content {
          include = instance_families.value.include
          exclude = instance_families.value.exclude
        }
      }

      dynamic "custom_priority" {
        for_each = constraints.value.custom_priority

        content {
          instance_families = custom_priority.value.instance_families
          on_demand         = custom_priority.value.on_demand
          spot              = custom_priority.value.spot
        }
      }

      dynamic "dedicated_node_affinity" {
        for_each = constraints.value.dedicated_node_affinity

        content {
          name           = dedicated_node_affinity.value.name
          az_name        = dedicated_node_affinity.value.az_name
          instance_types = dedicated_node_affinity.value.instance_types

          dynamic "affinity" {
            for_each = dedicated_node_affinity.value.affinity

            content {
              key      = affinity.value.key
              operator = affinity.value.operator
              values   = affinity.value.values
            }
          }
        }
      }

      dynamic "gpu" {
        for_each = constraints.value.gpu == null ? [] : [constraints.value.gpu]

        content {
          manufacturers = gpu.value.manufacturers
          include_names = gpu.value.include_names
          exclude_names = gpu.value.exclude_names
          min_count     = gpu.value.min_count
          max_count     = gpu.value.max_count
        }
      }

      dynamic "aws" {
        for_each = constraints.value.aws == null ? [] : [constraints.value.aws]

        content {
          dynamic "capacity_reservations" {
            for_each = aws.value.capacity_reservations

            content {
              id                          = capacity_reservations.value.id
              type                        = capacity_reservations.value.type
              capacity_resource_group_arn = capacity_reservations.value.capacity_resource_group_arn
            }
          }
        }
      }
    }
  }
}

resource "castai_autoscaler" "this" {
  count = var.autoscaler_settings == null ? 0 : 1

  cluster_id = castai_eks_cluster.this.id

  autoscaler_settings {
    enabled                                 = var.autoscaler_settings.enabled
    is_scoped_mode                          = var.autoscaler_settings.is_scoped_mode
    node_templates_partial_matching_enabled = var.autoscaler_settings.node_templates_partial_matching_enabled

    unschedulable_pods {
      enabled = var.autoscaler_settings.unschedulable_pods_enabled
    }

    cluster_limits {
      enabled = var.autoscaler_settings.cluster_limits_enabled

      cpu {
        min_cores = var.autoscaler_settings.cluster_limits_min_cpu
        max_cores = var.autoscaler_settings.cluster_limits_max_cpu
      }
    }

    node_downscaler {
      enabled = var.autoscaler_settings.node_downscaler_enabled

      empty_nodes {
        enabled = var.autoscaler_settings.empty_nodes_enabled
      }

      evictor {
        enabled         = var.autoscaler_settings.evictor_enabled
        aggressive_mode = var.autoscaler_settings.evictor_aggressive_mode
        cycle_interval  = var.autoscaler_settings.evictor_cycle_interval
        dry_run         = var.autoscaler_settings.evictor_dry_run

        node_grace_period_minutes = var.autoscaler_settings.evictor_node_grace_period
        scoped_mode               = var.autoscaler_settings.evictor_scoped_mode
      }
    }
  }
}
