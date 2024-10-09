
output "castai_cluster_id" {
  description = "ID of the CAST AI cluster"
  value       = module.castai-gke-cluster.cluster_id
}
