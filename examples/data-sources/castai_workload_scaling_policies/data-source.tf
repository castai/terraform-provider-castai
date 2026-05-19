# Fetch all scaling policies for the cluster.
data "castai_workload_scaling_policies" "cluster" {
  cluster_id = castai_gke_cluster.cluster.id
}

# policies_by_name provides a name -> ID map directly — no locals needed.
# policies list is also available for custom filtering with for expressions.

# Define managed policies as normal resources.
resource "castai_workload_scaling_policy" "my_policy" {
  cluster_id        = castai_gke_cluster.cluster.id
  name              = "my-policy"
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
    castai_workload_scaling_policy.my_policy.id,                                # managed — reference directly
    data.castai_workload_scaling_policies.cluster.policies_by_name["readonly"], # auto-generated castware policy
  ]
}
