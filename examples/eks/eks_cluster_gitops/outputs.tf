output "cluster_id" {
  value       = castai_eks_cluster.my_castai_cluster.id
  description = "CAST AI cluster ID"
}

output "cluster_token" {
  value       = castai_eks_cluster.my_castai_cluster.cluster_token
  description = "CAST AI cluster token used by Castware to atuhenticate to Mothership"
  sensitive   = true
}