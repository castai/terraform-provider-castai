output "instance_profile_role_arn" {
  value = module.castai-eks-role-iam.instance_profile_role_arn
}

output "role_arn" {
  value = module.castai-eks-role-iam.role_arn
}