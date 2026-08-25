resource "castai_autoscaler_policies" "policies" {
  cluster_id  = castai_eks_cluster.test.id
  enabled     = true
  scoped_mode = false

  cluster_limits {
    enabled = true

    cpu {
      max_cores = 10
    }
  }

  node_downscaler {
    empty_nodes_enabled = true
    empty_nodes_delay   = "90s"
  }

  unschedulable_pods {
    enabled = true

    pod_pinner {
      enabled = true
    }
  }
}
