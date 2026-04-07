# Pod mutation that adds spot scheduling and annotates pods in the "default" namespace.
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


  patch = jsonencode([
    {
      op    = "add"
      path  = "/metadata/annotations/mutated-by-pod-mutator"
      value = "true"
    }
  ])
}

# Pod mutation that assigns pods to a node template via node selector and toleration.
resource "castai_pod_mutation" "node_template_assignment" {
  cluster_id = castai_eks_cluster.example.id
  name       = "node-template-assignment"
  enabled    = true

  filter_v2 {
    workload {
      namespaces {
        type  = "REGEX"
        value = "^jobs-.*$"
      }
    }
  }

  patch = jsonencode([
    {
      op    = "add"
      path  = "/spec/nodeSelector/scheduling.cast.ai~1node-template"
      value = "jobs-nodes"
    },
    {
      op   = "add"
      path = "/spec/tolerations/-"
      value = {
        key      = "scheduling.cast.ai/node-template"
        value    = "jobs-nodes"
        effect   = "NoSchedule"
        operator = "Equal"
      }
    },
    {
      op    = "add"
      path  = "/metadata/annotations/cast.ai~1node-template"
      value = "jobs-nodes"
    }
  ])
}

# Pod mutation that migrates from cluster-autoscaler eviction annotation to CAST AI removal disabled.
resource "castai_pod_mutation" "eviction_annotation_migration" {
  cluster_id = castai_eks_cluster.example.id
  name       = "eviction-annotation-migration"
  enabled    = true

  filter_v2 {
    workload {
      namespaces {
        type  = "REGEX"
        value = ".*"
      }
    }
  }

  patch = jsonencode([
    {
      op    = "remove"
      path  = "/metadata/annotations/cluster-autoscaler.kubernetes.io~1safe-to-evict"
      value = "true"
    },
    {
      op    = "add"
      path  = "/metadata/annotations/autoscaling.cast.ai~1removal-disabled"
      value = "true"
    }
  ])
}

# Pod mutation with distribution groups, each routing pods to a different node pool via patches.
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
    name       = "spot-pool"
    percentage = 70
    configuration {
      spot_mode = "PREFERRED_SPOT"
      patch = jsonencode([
        {
          op    = "add"
          path  = "/spec/nodeSelector/agentpool"
          value = "spot"
        }
      ])
    }
  }

  distribution_groups {
    name       = "on-demand-pool"
    percentage = 30
    configuration {
      spot_mode = "OPTIONAL_SPOT"
      patch = jsonencode([
        {
          op    = "add"
          path  = "/spec/nodeSelector/agentpool"
          value = "common"
        }
      ])
    }
  }
}
