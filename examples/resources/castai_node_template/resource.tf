resource "castai_node_template" "default_by_castai" {
  cluster_id = castai_eks_cluster.test.id

  name             = "default-by-castai"
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
    azs                                         = ["us-east-2a", "us-east-2b"]
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
