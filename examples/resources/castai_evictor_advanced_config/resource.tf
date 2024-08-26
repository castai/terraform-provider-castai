resource "castai_evictor_advanced_config" "config" {
  cluster_id = castai_eks_cluster.test.id
  evictor_advanced_config {
    pod_selector {
      kind      = "Job"
      namespace = "test"
      match_labels = {
        "job" = "test"
      }
    }
    aggressive = true
  }
}
