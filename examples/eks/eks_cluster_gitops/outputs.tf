output "cluster_id" {
  value       = castai_eks_clusterid.cluster_id.id
  description = "CAST AI cluster ID"
}

output "instance_profile_role_arn" {
  description = "Arn of created cast instance role"
  value       = module.castai-eks-role-iam.instance_profile_role_arn
}


output "instance_profile_arn" {
  description = "Arn of created cast instanceprofile role"
  value       = module.castai-eks-role-iam.instance_profile_arn
}


output "cast_role_arn" {
  description = "Arn of created cast role"
  value       = module.castai-eks-role-iam.role_arn
}
