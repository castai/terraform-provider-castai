# Fetch all scaling policies for the cluster.
data "castai_workload_scaling_policies" "cluster" {
  cluster_id = castai_gke_cluster.cluster.id
}

# Build a name -> ID map for easy lookup.
locals {
  policy_by_name = { for p in data.castai_workload_scaling_policies.cluster.policies : p.name => p.id }
}

# Define managed policies as normal resources.
resource "castai_workload_scaling_policy" "htz_balanced" {
  cluster_id        = castai_gke_cluster.cluster.id
  name              = "htz-balanced"
  apply_type        = "IMMEDIATE"
  management_option = "MANAGED"
  cpu {
    function = "QUANTILE"
    args     = ["0.9"]
    overhead = 0.1
  }
  memory {
    function = "MAX"
    overhead = 0.1
  }
}

# Use direct resource references for managed policies and the name map
# for auto-generated castware policies — no hardcoded UUIDs anywhere.
resource "castai_workload_scaling_policy_order" "custom" {
  cluster_id = castai_gke_cluster.cluster.id
  policy_ids = [
    castai_workload_scaling_policy.htz_balanced.id, # managed — reference directly
    local.policy_by_name["readonly"],               # auto-generated castware policy
  ]
}
