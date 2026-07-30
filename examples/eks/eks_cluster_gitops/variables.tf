## Required variables.

variable "aws_account_id" {
  type        = string
  description = "ID of AWS account the cluster is located in."
}

variable "aws_cluster_region" {
  type        = string
  description = "Region of the cluster to be connected to CAST AI."
}

variable "aws_cluster_name" {
  type        = string
  description = "Name of the cluster to be connected to CAST AI."
}

variable "castai_api_token" {
  type        = string
  sensitive   = true
  description = "CAST AI API token created in console.cast.ai API Access keys section"
}

variable "subnets" {
  type        = list(string)
  description = "Subnet IDs used by CAST AI to provision nodes. Used as the default for any node configuration that does not set its own `subnets`."
}

variable "security_group_ids" {
  type        = list(string)
  description = "Security group IDs (usually the EKS cluster and node security groups) attached to CAST AI provisioned nodes. Used as the default for any node configuration that does not set its own `eks.security_groups`."
}

variable "vpc_id" {
  type        = string
  description = "EKS cluster VPC ID"
}

## Optional variables.

variable "profile" {
  type        = string
  description = "Profile used with AWS CLI"
  default     = "default"
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI url to API, default value is https://api.cast.ai"
  default     = "https://api.cast.ai"
}

variable "delete_nodes_on_disconnect" {
  type        = bool
  description = "Optionally delete CAST AI created nodes when the cluster is destroyed"
  default     = false
}

## Node configurations.

variable "node_configurations" {
  description = <<-EOT
    Node configurations to create, keyed by configuration name. Every attribute is optional;
    omitted attributes fall back to the CAST AI defaults, except for `subnets` and
    `eks.security_groups` / `eks.instance_profile_arn` which fall back to `var.subnets`,
    `var.security_group_ids` and the instance profile created by the castai-eks-role-iam module.
  EOT

  type = map(object({
    disk_cpu_ratio    = optional(number)
    min_disk_size     = optional(number)
    drain_timeout_sec = optional(number)
    subnets           = optional(list(string))
    ssh_public_key    = optional(string)
    image             = optional(string)
    init_script       = optional(string)
    container_runtime = optional(string)
    docker_config     = optional(string)
    kubelet_config    = optional(string)
    tags              = optional(map(string))

    eks = optional(object({
      security_groups               = optional(list(string))
      instance_profile_arn          = optional(string)
      node_group_arn                = optional(string)
      dns_cluster_ip                = optional(string)
      key_pair_id                   = optional(string)
      volume_type                   = optional(string)
      volume_iops                   = optional(number)
      volume_throughput             = optional(number)
      volume_kms_key_arn            = optional(string)
      imds_v1                       = optional(bool)
      imds_hop_limit                = optional(number)
      max_pods_per_node_formula     = optional(string)
      ips_per_prefix                = optional(number)
      threads_per_cpu               = optional(number)
      eks_image_family              = optional(string)
      ena_queue_count_per_interface = optional(number)

      target_group = optional(list(object({
        arn  = string
        port = optional(number)
      })), [])
    }), {})
  }))

  default = {
    default = {
      disk_cpu_ratio = 0
      min_disk_size  = 100
    }
  }

  validation {
    condition     = length(var.node_configurations) > 0
    error_message = "At least one node configuration must be defined."
  }
}

variable "default_node_configuration" {
  type        = string
  description = "Key of the entry in `var.node_configurations` that is promoted to the cluster's default node configuration."
  default     = "default"
}

## Node templates.

variable "node_templates" {
  description = <<-EOT
    Node templates to create, keyed by template name. `node_configuration` references a key of
    `var.node_configurations`; when omitted, the template is attached to
    `var.default_node_configuration`.
  EOT

  type = map(object({
    node_configuration           = optional(string)
    is_default                   = optional(bool, false)
    is_enabled                   = optional(bool, true)
    should_taint                 = optional(bool, true)
    custom_labels                = optional(map(string), {})
    custom_instances_enabled     = optional(bool)
    rebalancing_config_min_nodes = optional(number)

    custom_taints = optional(list(object({
      key    = string
      value  = optional(string)
      effect = optional(string, "NoSchedule")
    })), [])

    constraints = optional(object({
      on_demand                                   = optional(bool)
      spot                                        = optional(bool)
      use_spot_fallbacks                          = optional(bool)
      fallback_restore_rate_seconds               = optional(number)
      enable_spot_diversity                       = optional(bool)
      spot_diversity_price_increase_limit_percent = optional(number)
      spot_interruption_predictions_enabled       = optional(bool)
      spot_interruption_predictions_type          = optional(string)
      spot_reliability_enabled                    = optional(bool)
      compute_optimized_state                     = optional(string)
      storage_optimized_state                     = optional(string)
      is_gpu_only                                 = optional(bool)
      min_cpu                                     = optional(number)
      max_cpu                                     = optional(number)
      min_memory                                  = optional(number)
      max_memory                                  = optional(number)
      architectures                               = optional(list(string))
      os                                          = optional(list(string))
      azs                                         = optional(list(string))
      burstable_instances                         = optional(string)
      customer_specific                           = optional(string)
      bare_metal                                  = optional(string)
      cpu_manufacturers                           = optional(list(string))

      instance_families = optional(object({
        include = optional(list(string), [])
        exclude = optional(list(string), [])
      }))

      custom_priority = optional(list(object({
        instance_families = optional(list(string), [])
        on_demand         = optional(bool)
        spot              = optional(bool)
      })), [])

      dedicated_node_affinity = optional(list(object({
        name           = string
        az_name        = string
        instance_types = optional(list(string), [])
        affinity = optional(list(object({
          key      = string
          operator = string
          values   = list(string)
        })), [])
      })), [])

      gpu = optional(object({
        manufacturers = optional(list(string))
        include_names = optional(list(string))
        exclude_names = optional(list(string))
        min_count     = optional(number)
        max_count     = optional(number)
      }))

      aws = optional(object({
        capacity_reservations = optional(list(object({
          id                          = optional(string)
          type                        = optional(string)
          capacity_resource_group_arn = optional(string)
        })), [])
      }))
    }))
  }))

  default = {
    default-by-castai = {
      is_default   = true
      should_taint = false
      constraints = {
        on_demand = true
      }
    }
  }
}

## Autoscaler.

variable "autoscaler_settings" {
  description = "CAST AI autoscaler settings. Set to null to skip managing the castai_autoscaler resource."

  type = object({
    enabled                                 = optional(bool, true)
    is_scoped_mode                          = optional(bool, false)
    node_templates_partial_matching_enabled = optional(bool, false)

    unschedulable_pods_enabled = optional(bool, true)

    cluster_limits_enabled = optional(bool, false)
    cluster_limits_min_cpu = optional(number, 1)
    cluster_limits_max_cpu = optional(number, 200)

    node_downscaler_enabled   = optional(bool, true)
    empty_nodes_enabled       = optional(bool, true)
    evictor_enabled           = optional(bool, true)
    evictor_aggressive_mode   = optional(bool, false)
    evictor_cycle_interval    = optional(string, "60s")
    evictor_dry_run           = optional(bool, false)
    evictor_node_grace_period = optional(number, 10)
    evictor_scoped_mode       = optional(bool, false)
  })

  default = {}
}
