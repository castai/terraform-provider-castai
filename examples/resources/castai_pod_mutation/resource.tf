# Basic pod mutation that adds spot scheduling to Deployments in the "default" namespace.
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

  tolerations {
    key      = "scheduling.cast.ai/spot"
    operator = "Exists"
    effect   = "NoSchedule"
  }
}

# Apply spot to all workloads cluster-wide, excluding system namespaces.
# A common pattern to maximize spot usage while keeping infrastructure workloads on on-demand nodes.
resource "castai_pod_mutation" "spot_all_except_system" {
  cluster_id = castai_eks_cluster.example.id
  name       = "spot-all-except-system"
  enabled    = true

  filter_v2 {
    workload {
      exclude_namespaces {
        type  = "EXACT"
        value = "kube-system"
      }
      exclude_namespaces {
        type  = "EXACT"
        value = "castai-agent"
      }
    }
  }

  spot_config {
    spot_mode               = "USE_ONLY_SPOT"
    distribution_percentage = 100
  }
}

# Apply spot to all workloads, excluding DaemonSets and StatefulSets by kind,
# and excluding specific infrastructure workloads by name.
resource "castai_pod_mutation" "spot_exclude_infra" {
  cluster_id = castai_eks_cluster.example.id
  name       = "spot-exclude-infra"
  enabled    = true

  filter_v2 {
    workload {
      exclude_kinds {
        type  = "EXACT"
        value = "DaemonSet"
      }
      exclude_kinds {
        type  = "EXACT"
        value = "StatefulSet"
      }
      exclude_names {
        type  = "EXACT"
        value = "castai-agent"
      }
      exclude_names {
        type  = "EXACT"
        value = "castai-cluster-controller"
      }
    }
  }

  spot_config {
    spot_mode               = "USE_ONLY_SPOT"
    distribution_percentage = 100
  }
}

# Target specific workloads by name to place them on dedicated nodes
# using node selectors and tolerations.
resource "castai_pod_mutation" "dedicated_nodes_by_name" {
  cluster_id = castai_eks_cluster.example.id
  name       = "dedicated-nodes-by-name"
  enabled    = true

  filter_v2 {
    workload {
      names {
        type  = "EXACT"
        value = "my-redis-cluster-prod"
      }
      names {
        type  = "EXACT"
        value = "my-redis-cache-prod"
      }
    }
  }

  node_selector {
    add = {
      "dedicated" = "redis"
    }
  }

  tolerations {
    key      = "dedicated"
    value    = "redis"
    operator = "Equal"
    effect   = "NoSchedule"
  }
}

# Apply a node template to Argo Rollouts across the cluster,
# excluding specific namespaces that should remain on on-demand nodes.
resource "castai_pod_mutation" "rollouts_node_template" {
  cluster_id = castai_eks_cluster.example.id
  name       = "rollouts-node-template"
  enabled    = true

  filter_v2 {
    workload {
      kinds {
        type  = "EXACT"
        value = "Rollout"
      }
      exclude_namespaces {
        type  = "EXACT"
        value = "infra"
      }
      exclude_namespaces {
        type  = "EXACT"
        value = "monitoring"
      }
    }
  }

  spot_config {
    spot_mode               = "PREFERRED_SPOT"
    distribution_percentage = 100
  }

  node_templates_to_consolidate = ["default-by-castai"]
}

# Pod mutation with node affinity to prefer spot-capable nodes in specific availability zones,
# combined with a toleration to allow scheduling on spot nodes.
resource "castai_pod_mutation" "spot_with_affinity" {
  cluster_id = castai_eks_cluster.example.id
  name       = "spot-with-affinity"
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
    distribution_percentage = 100
  }

  affinity {
    node_affinity {
      preferred_during_scheduling_ignored_during_execution {
        weight = 1
        preference {
          match_expressions {
            key      = "topology.kubernetes.io/zone"
            operator = "In"
            values   = ["us-east-1a", "us-east-1b"]
          }
        }
      }
    }
  }

  tolerations {
    key      = "scheduling.cast.ai/spot"
    operator = "Exists"
    effect   = "NoSchedule"
  }
}

# Pod mutation using raw JSON patches to add annotations and resource limits
# to web workloads matched by name prefix, excluding a specific workload.
resource "castai_pod_mutation" "json_patches" {
  cluster_id = castai_eks_cluster.example.id
  name       = "json-patches"
  enabled    = true

  filter_v2 {
    workload {
      namespaces {
        type  = "EXACT"
        value = "default"
      }
      names {
        type  = "REGEX"
        value = "^web-.*$"
      }
      exclude_names {
        type  = "EXACT"
        value = "web-internal"
      }
    }
  }

  patch = jsonencode([
    {
      op    = "add"
      path  = "/metadata/annotations/mutated-by"
      value = "castai"
    },
    {
      op    = "add"
      path  = "/spec/containers/0/resources/limits/memory"
      value = "512Mi"
    },
  ])
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
      exclude_namespaces {
        type  = "EXACT"
        value = "prod-legacy"
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
