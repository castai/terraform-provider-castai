resource "castai_evictor_advanced_config" "config" {
  cluster_id = castai_eks_cluster.test.id
  evictor_advanced_config {
    pod_selector {
      kind         = "Deployment"
      namespace    = "test"
      replicas_min = 2
      match_labels = {
        "app" = "test"
      }
    }
    aggressive = true
  }
}
