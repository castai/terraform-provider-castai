# Basic pod mutation that adds spot scheduling to workloads in the "default" namespace.
resource "castai_pod_mutation" "spot_scheduling" {
  cluster_id = castai_eks_cluster.example.id
  name       = "spot-scheduling"
  enabled    = true

  filter_v2 {
    workload {
      namespaces {
        type  = "EXACT"
        value = "default"
      }
      kinds {
        type  = "EXACT"
        value = "Deployment"
      }
    }
  }

  spot_config {
    spot_mode               = "PREFERRED_SPOT"
    distribution_percentage = 80
  }
  restart_matching_workloads   = true

  tolerations {
    key      = "scheduling.cast.ai/spot"
    operator = "Exists"
    effect   = "NoSchedule"
  }
}

# Pod mutation with distribution groups to split pods across different node pools
# with distinct configurations (tolerations, node selectors, node templates).
resource "castai_pod_mutation" "multi_pool_distribution" {
  cluster_id = castai_eks_cluster.example.id
  name       = "multi-pool-distribution"
  enabled    = true

  filter_v2 {
    workload {
      namespaces {
        type  = "REGEX"
        value = "^prod-.*$"
      }
    }
    pod {
      labels_filter {
        operator = "AND"
        matchers {
          key {
            type  = "EXACT"
            value = "app"
          }
          value {
            type  = "EXACT"
            value = "web"
          }
        }
      }
    }
  }

  distribution_groups {
    name       = "gpu-pool"
    percentage = 30
    configuration {
      spot_type = "PREFERRED_SPOT"
      tolerations {
        key      = "nvidia.com/gpu"
        operator = "Exists"
        effect   = "NoSchedule"
      }
      node_selector {
        add = {
          "node.kubernetes.io/gpu" = "true"
        }
      }
    }
  }

  distribution_groups {
    name       = "general-pool"
    percentage = 70
    configuration {
      spot_type                     = "OPTIONAL_SPOT"
      node_templates_to_consolidate = ["default-general"]
    }
  }
}
