output "eks_cluster_authentication_mode" {
  value = data.aws_eks_cluster.existing_cluster.access_config[0].authentication_mode
}
