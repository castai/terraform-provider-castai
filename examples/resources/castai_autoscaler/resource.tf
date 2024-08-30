resource "castai_autoscaler" "castai_autoscaler_policy" {
  cluster_id = castai_eks_cluster.test.id

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
        cycle_interval                         = "300s"
        node_grace_period_minutes              = 10
        pod_eviction_failure_back_off_interval = "30s"
        ignore_pod_disruption_budgets          = false
      }
    }
  }
}
