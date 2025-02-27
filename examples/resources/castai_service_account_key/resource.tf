resource "castai_service_account_key" "service_account_key" {
  organization_id    = data.castai_organization.test.id
  service_account_id = castai_service_account.service_account.id
  name               = "example-key"
  active             = true
  expires_at         = "2026-01-01T00:00:00Z"
}

output "service_account_key" {
  value = castai_service_account_key.service_account_key.token
}
