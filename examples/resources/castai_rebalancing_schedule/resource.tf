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
    keep_drain_timeout_nodes = true
    # When keep_drain_timeout_nodes is true, configure how drain-failed nodes are handled.
    drain_failure_config {
      # Set to true to leave drain-failed nodes cordoned indefinitely (no auto-uncordon).
      disable_uncordon = false
      # Uncordon drain-failed nodes after 2 hours (default is 30m; range: 1m–72h).
      uncordon_after_seconds = 7200
    }
    # equivalent to the deprecated config: aggressive_mode = true.
    aggressive_mode_config {
      ignore_local_persistent_volumes        = true
      ignore_problem_job_pods                = true
      ignore_problem_pods_without_controller = true
      ignore_problem_removal_disabled_pods   = true
    }
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
