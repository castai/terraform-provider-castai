output "castai_storage_autoscaler_role_arn" {
  description = "IRSA role assumed by the castai-storage-autoscaler Kubernetes service account."
  value       = aws_iam_role.castai_storage_autoscaler.arn
}

output "castai_kvisor_role_arn" {
  description = "IRSA role assumed by the castai-kvisor-controller Kubernetes service account."
  value       = aws_iam_role.castai_kvisor.arn
}
