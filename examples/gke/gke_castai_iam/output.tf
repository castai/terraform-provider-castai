output "private_key" {
  value     = module.castai-gke-iam.private_key
  sensitive = true
}

output "service_account_id" {
  value = module.castai-gke-iam.service_account_id
}

output "service_account_email" {
  value = module.castai-gke-iam.service_account_email
}

output "castai_service_account_id" {
  value = module.castai-gke-iam.castai_service_account_id
}

output "castai_service_account_email" {
  value = module.castai-gke-iam.castai_service_account_email
}

output "target_service_account_email" {
  value = module.castai-gke-iam.target_service_account_email
}
