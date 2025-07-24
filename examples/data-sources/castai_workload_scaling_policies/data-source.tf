data "castai_workload_scaling_policies" "cluster" {
  cluster_id = castai_gke_cluster.cluster.id
}
