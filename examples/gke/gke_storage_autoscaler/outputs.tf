output "castai_storage_autoscaler_service_account_email" {
  description = "GCP service account used by the castai-storage-autoscaler Kubernetes service account."
  value       = google_service_account.castai_storage_autoscaler.email
}

output "castai_kvisor_service_account_email" {
  description = "GCP service account used by the castai-kvisor-controller Kubernetes service account."
  value       = google_service_account.castai_kvisor.email
}
