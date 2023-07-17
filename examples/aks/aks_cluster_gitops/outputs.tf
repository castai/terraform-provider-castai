output "cluster_id" {
  value       = castai_aks_cluster.this.id
  description = "CAST AI cluster ID."
}

output "cluster_token" {
  value       = castai_aks_cluster.this.cluster_token
  description = "CAST AI cluster token used by Castware to atuhenticate to Mothership."
  sensitive   = true
}
