resource "evictor_advanced_config" "config" {
  evictor_advanced_config = [
    {
      pod_selector = {
        kind      = "Job"
        namespace = "castai"
        match_labels = {
          "app.kubernetes.io/name" = "castai-node"
        }
      },
      aggressive = true
    },
    {
      node_selector = {
        match_expressions = [
          {
            key      = "pod.cast.ai/flag"
            operator = "Exists"
          }
        ]
      },
      disposable = true
    }
  ]
}
