data "castai_workload_scaling_policies" "cluster" {
  cluster_id = castai_gke_cluster.cluster.id
}

locals {
  ordered_policy_ids = [
    "be61e44b-0f7c-44da-b60e-0594d6fd3634",
    "f60b79f0-d21e-4eda-a59f-f5daa846d289",
  ]
  unordered_policy_ids = tolist(setsubtract(data.castai_workload_scaling_policies.cluster.policy_ids, local.ordered_policy_ids))
}

resource "castai_workload_scaling_policy_order" "custom" {
  cluster_id = castai_gke_cluster.cluster.id
  policy_ids = concat(local.ordered_policy_ids, sort(local.unordered_policy_ids))
}
