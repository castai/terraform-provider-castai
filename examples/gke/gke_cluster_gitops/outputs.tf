output "cluster_id" {
  value       = castai_gke_cluster.this.id
  description = "CAST AI cluster ID."
}

output "cluster_token" {
  value       = castai_gke_cluster.this.cluster_token
  description = "CAST AI cluster token used by Castware to authenticate to Mothership."
  sensitive   = true
}
