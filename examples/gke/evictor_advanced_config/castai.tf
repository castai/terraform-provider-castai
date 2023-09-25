provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

module "gke_autoscaler_evictor_advanced_config" {
  source = "../gke_cluster_autoscaler_policies"

  castai_api_token           = var.castai_api_token
  castai_api_url             = var.castai_api_url
  cluster_region             = var.cluster_region
  cluster_zones              = var.cluster_zones
  cluster_name               = var.cluster_name
  project_id                 = var.project_id
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect
  evictor_advanced_config    = [
    {
      pod_selector = {
        kind         = "Job"
        namespace    = "castai"
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