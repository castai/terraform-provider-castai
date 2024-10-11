
output "castai_cluster_id" {
  description = "ID of the CAST AI cluster"
  value       = castai_gke_cluster_id.this.id
}
