output "sgs" {
  description = "EKS cluster security groups"
  value = module.castai-eks-cluster.security_groups
}
