output "castai_cluster_id" {
  value     = module.castai-gke-cluster.cluster_id
  sensitive = true
}
