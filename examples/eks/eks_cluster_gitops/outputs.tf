output "cluster_id" {
  value       = castai_eks_clusterid.cluster_id.id
  description = "CAST AI cluster ID"
}

output "cluster_token" {
  value       = castai_eks_cluster.this.cluster_token
  description = "CAST AI cluster token used by Castware to authenticate to Mothership"
  sensitive   = true
}

output "instance_profile_role_arn" {
  description = "Arn of created cast instance role"
  value       = module.castai-eks-role-iam.instance_profile_role_arn
}

output "instance_profile_arn" {
  description = "Arn of created cast instance profile role"
  value       = module.castai-eks-role-iam.instance_profile_arn
}

output "cast_role_arn" {
  description = "Arn of created cast role"
  value       = module.castai-eks-role-iam.role_arn
}

output "node_configuration_ids" {
  description = "Map of node configuration name to CAST AI node configuration ID"
  value       = { for name, cfg in castai_node_configuration.this : name => cfg.id }
}

output "node_template_names" {
  description = "Names of the node templates managed by this module"
  value       = keys(castai_node_template.this)
}
