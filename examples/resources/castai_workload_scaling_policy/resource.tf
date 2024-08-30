castai_workload_scaling_policy "services" {
  name              = "services"
  cluster_id        = castai_gke_cluster.dev.id
  apply_type        = "IMMEDIATE"
  management_option = "MANAGED"
  cpu {
    function        = "QUANTILE"
    overhead        = 0.15
    apply_threshold = 0.1
    args            = ["0.9"]
  }
  memory {
    function        = "MAX"
    overhead        = 0.35
    apply_threshold = 0.2
  }
}