resource "castai_autoscaler" "castai_autoscaler_policy" {
  cluster_id = castai_eks_cluster.castai_cluster.id

  autoscaler_policies_json = var.autoscaler_policies_json

  autoscaler_settings {
    enabled                                 = true
    is_scoped_mode                          = false
    node_templates_partial_matching_enabled = false

    unschedulable_pods {
      enabled                  = true
      custom_instances_enabled = false

      headroom {
        enabled           = true
        cpu_percentage    = 20
        memory_percentage = 30
      }

      headroom_spot {
        enabled           = true
        cpu_percentage    = 15
        memory_percentage = 25
      }

      node_constraints {
        enabled       = true
        min_cpu_cores = 2
        max_cpu_cores = 8
        min_ram_mib   = 2048
        max_ram_mib   = 8192
      }
    }

    cluster_limits {
      enabled = true

      cpu {
        min_cores = 1
        max_cores = 10
      }
    }

    spot_instances {
      enabled                             = true
      max_reclaim_rate                    = 50
      spot_diversity_enabled              = true
      spot_diversity_price_increase_limit = 20

      spot_backups {
        enabled                          = true
        spot_backup_restore_rate_seconds = 300
      }

      spot_interruption_predictions {
        enabled                            = true
        spot_interruption_predictions_type = "history"
      }
    }

    node_downscaler {
      enabled = true

      empty_nodes {
        enabled       = true
        delay_seconds = 60
      }

      evictor {
        enabled                                = true
        dry_run                                = false
        aggressive_mode                        = false
        scoped_mode                            = false
        cycle_interval                         = 300
        node_grace_period_minutes              = 10
        pod_eviction_failure_back_off_interval = 30
        ignore_pod_disruption_budgets          = false
      }
    }
  }
}
