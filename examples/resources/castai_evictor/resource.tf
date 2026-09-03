resource "castai_evictor" "evictor" {
  cluster_id = castai_eks_cluster.test.id

  enabled                                = true
  dry_run                                = false
  aggressive_mode                        = false
  scoped_mode                            = false
  cycle_interval                         = "60s"
  node_grace_period_minutes              = 10
  pod_eviction_failure_back_off_interval = "30s"
  ignore_pod_disruption_budgets          = false
  soft_tainting                          = false
  emit_node_related_pod_events           = false

  # The fields below are exposed in the V2 API proto and are settable in
  # Terraform. They may not be active in the backend yet and will become
  # functional as backend support rolls out.
  drain_timeout                = "10m"
  drain_rollback_timeout       = "1m"
  windows                      = false
  force_disable_live_migration = false
  force_disable_woop           = false
  force_disable_pod_mutations  = false
  force_disable_karpenter_mode = false
  max_target_nodes_per_cycle   = 0
  min_target_nodes_per_cycle   = 0
  target_node_percentage       = 0
  pricing_awareness_enabled    = false
  arm64_supported              = false

  pricing_model {
    base_cpu_cost = "0.5"
    base_mem_cost = "0.5"
    spot_discount = "0.5"
  }
}
