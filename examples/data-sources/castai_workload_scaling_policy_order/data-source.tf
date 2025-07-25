data "castai_workload_scaling_policy_order" "cluster" {
  cluster_id = castai_gke_cluster.cluster.id
}
