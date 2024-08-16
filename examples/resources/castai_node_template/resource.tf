resource "castai_node_template" "this" {
  cluster_id = castai_eks_cluster.castai_cluster.id

  name             = "my-node-template"
  is_default       = false
  is_enabled       = true
  configuration_id = "config-id-123"
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
    compute_optimized                           = true
    storage_optimized                           = false
    compute_optimized_state                     = "on"
    storage_optimized_state                     = "off"
    is_gpu_only                                 = false
    spot                                        = true
    on_demand                                   = false
    use_spot_fallbacks                          = true
    fallback_restore_rate_seconds               = 300
    enable_spot_diversity                       = true
    spot_diversity_price_increase_limit_percent = 20
    spot_interruption_predictions_enabled       = true
    spot_interruption_predictions_type          = "history"
    min_cpu                                     = 2
    max_cpu                                     = 8
    min_memory                                  = 4096
    max_memory                                  = 16384
    architectures                               = ["amd64"]
    azs                                         = ["us-east-1a", "us-east-1b"]
    burstable_instances                         = false
    customer_specific                           = false

    instance_families {
      include = ["m5", "m6i"]
      exclude = ["m4"]
    }

    gpu {
      manufacturers = ["nvidia"]
      include_names = ["p2"]
      exclude_names = ["p3"]
      min_count     = 1
      max_count     = 4
    }

    custom_priority {
      instance_families = ["m5", "m6i"]
      spot              = true
      on_demand         = false
    }
  }

  depends_on = [castai_autoscaler.castai_autoscaler_policies]
}
